package match

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/jakubDoka/keeper/index"
	"github.com/jakubDoka/keeper/knet"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util/uuid"
)

// Match holds basic state maintaining the player connections. Operations on
// match are not thread safe unless stated otherwise. The match state is maintained
// by a look and it uses a tick frequency to control how often it updates.
type Match struct {
	Core

	id, creator uuid.UUID
	state       *state.State
	tag         []index.Field
	manager     *Manager

	users      map[uuid.UUID]User
	idBuffer   []uuid.UUID
	userAmount uint32

	queuedUsers, tempQueuedUsers []User
	queuedUsersMutex             sync.Mutex

	tickRate   int
	ticker     *time.Ticker
	terminated bool
}

// New constructs a new match. meta is passed to core.OnInit method.
func New(state *state.State, manager *Manager, core Core, creator *state.User, id uuid.UUID, meta []byte) (*Match, error) {
	if id == uuid.Nil {
		id = uuid.New()
	}

	m := &Match{
		id:       id,
		Core:     core,
		creator:  creator.ID(),
		state:    state,
		manager:  manager,
		users:    make(map[uuid.UUID]User),
		tickRate: 30,
		ticker:   time.NewTicker(time.Second / 30),
	}

	err := core.OnInit(m.State(), meta)

	return m, err
}

func (m *Match) Info() []byte {
	inf, err := m.OnInfoRequest(m.State())
	if err != nil {
		m.state.Error("Error while getting match info: %s", err)
		return nil
	}
	return inf
}

// ConnectUser is thread safe and you can call it from anywhere. Match will not handle
// the connection immediately though.
func (m *Match) ConnectUser(user *state.User, conn *knet.Connection, meta []byte) {
	m.queuedUsersMutex.Lock()
	m.queuedUsers = append(m.queuedUsers, User{user, conn, meta})
	m.queuedUsersMutex.Unlock()
}

// Run launches a match main loop. This should be run on goroutine.
func (m *Match) Run() {
	var buffer []knet.ClientPacket
	var requests []Request
	var helper [][]byte

	state := m.State()

	for !m.terminated {
		userAmount := uint32(len(m.users))
		// handle disconnected and custom requests
		for id, user := range m.users {
			if user.Disconnected() {
				if m.handleErr(m.OnDisconnection(state, user)) {
					return
				}
				user.Close()
				delete(m.users, id)
			} else {
				requests = requests[:0]
				buffer = buffer[:0]

				user.HarvestPackets(m.state, &buffer, &helper)
				for _, packet := range buffer {
					requests = append(requests, Request{user.Connection, packet})
				}

				if m.handleErr(m.OnCustomRequest(state, requests)) {
					return
				}
			}
		}

		// handle incoming
		m.queuedUsersMutex.Lock()
		m.queuedUsers, m.tempQueuedUsers = m.tempQueuedUsers[:0], m.queuedUsers
		m.queuedUsersMutex.Unlock()
		for _, user := range m.tempQueuedUsers {
			meta, err, fatalErr := m.OnConnection(state, user, user.meta)

			if m.handleErr(fatalErr) {
				return
			}

			user.meta = nil

			if err != nil {
				user.WritePacketTCP(knet.OCMatchJoinFail, []byte(err.Error()))
			} else {
				user.WritePacketTCP(knet.OCMatchJoinSuccess, meta)
				m.users[user.User.ID()] = user
			}
		}

		// handle tick
		if m.handleErr(m.OnTick(state)) {
			return
		}

		newUserAmount := uint32(len(m.users))
		if newUserAmount != userAmount {
			atomic.StoreUint32(&m.userAmount, newUserAmount)
		}

		<-m.ticker.C
	}

	m.OnEnd(state)
	state.Debug("Match %s terminated", m.id)
}

func (m *Match) UserAmount() uint32 {
	return atomic.LoadUint32(&m.userAmount)
}

// ResendPacket just passes the packet as user intended to send it.
func (m *Match) ResendPacket(request knet.ClientPacket) {
	m.SendPacket(&request.Targets, request.OpCode, request.Data, request.Udp)
}

// Send packet sends a packet to all targets. Nil means all players.
func (m *Match) SendPacket(targets *[]uuid.UUID, opCode knet.OpCode, data []byte, udp bool) {
	if targets != nil {
		if len(*targets) == 0 {
			return
		}
		for _, target := range *targets {
			user, ok := m.users[target]
			if ok {
				user.WritePacket(opCode, data, udp)
			}
		}
	} else {
		for _, user := range m.users {
			user.WritePacket(opCode, data, udp)
		}
	}
}

// SetTickRate changes a tickrate of match
func (m *Match) SetTickRate(rate int) {
	if rate == 0 {
		panic("tick rate cannot be 0")
	}

	m.tickRate = rate

	duration := 1 * time.Second / time.Duration(rate)

	m.ticker.Reset(duration)
}

func (m *Match) SetTag(value []byte) (int, error) {
	var parser index.Parser
	tag, i, err := parser.Parse(value)
	if err != nil {
		return i, err
	}
	for i := range tag {
		tag[i].Value = &m.id
	}
	m.manager.index.Remove(m.tag...)
	m.tag = tag
	m.manager.index.Insert(m.tag...)
	return 0, nil
}

func (m *Match) GetUser(id uuid.UUID) (User, bool) {
	user, ok := m.users[id]
	return user, ok
}

func (m *Match) handleErr(err error) bool {
	if err == nil {
		return false
	}

	res := m.OnError(m.State(), err)
	if res {
		m.OnEnd(m.State())
	}
	return res
}

func (m *Match) Terminate() {
	m.terminated = true
}

// State return match state.
func (m *Match) State() State {
	return State{m.state, m}
}

func (m *Match) ID() uuid.UUID {
	return m.id
}

func (m *Match) Creator() uuid.UUID {
	return m.creator
}

func (m *Match) Only(target uuid.UUID) *[]uuid.UUID {
	m.idBuffer = m.idBuffer[:0]
	m.idBuffer = append(m.idBuffer, target)
	return &m.idBuffer
}

// All should be passed to SendPacket when sending to all players.
func (m *Match) All() *[]uuid.UUID {
	return nil
}

// AllExcept should be passed to SendPacket if sending packet to all but one player
func (m *Match) AllExcept(except uuid.UUID) *[]uuid.UUID {
	m.idBuffer = m.idBuffer[:0]
	for id := range m.users {
		if id != except {
			m.idBuffer = append(m.idBuffer, id)
		}
	}
	return &m.idBuffer
}

// User is match side extension of state.User.
type User struct {
	*state.User
	*knet.Connection
	meta []byte
}

// State is match extension of state.State.
type State struct {
	*state.State
	*Match
}

func (s State) UserAmount() int {
	return len(s.users)
}

// Request glues packet and sender together
type Request struct {
	*knet.Connection
	knet.ClientPacket
}

type Core interface {
	OnInit(state State, meta []byte) error
	OnConnection(state State, user User, meta []byte) ([]byte, error, error)
	OnDisconnection(state State, user User) error
	OnCustomRequest(state State, req []Request) error
	OnTick(state State) error
	OnError(state State, err error) bool
	OnEnd(state State)
	OnInfoRequest(state State) ([]byte, error)
}

type CoreBase struct{}

func (*CoreBase) OnInit(state State, meta []byte) error { return nil }
func (*CoreBase) OnConnection(state State, user User, meta []byte) ([]byte, error, error) {
	return nil, nil, nil
}
func (*CoreBase) OnDisconnection(state State, req User) error      { return nil }
func (*CoreBase) OnCustomRequest(state State, req []Request) error { return nil }
func (*CoreBase) OnTick(state State) error                         { return nil }
func (*CoreBase) OnError(state State, err error) bool              { return true }
func (*CoreBase) OnEnd(state State)                                {}
func (*CoreBase) OnInfoRequest(state State) ([]byte, error)        { return nil, nil }
