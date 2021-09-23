package match

import (
	"sync"
	"time"

	"github.com/jakubDoka/keeper/knet"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util/uuid"
)

// Match holds basic state maintaining the player connections. Operations on
// match are not thread safe unless stated otherwise. The match state is maintained
// by a look and it uses a tick frequency to control how often it updates.
type Match struct {
	Core

	Id      uuid.UUID
	Creator *state.User

	state *state.State

	users    map[uuid.UUID]User
	idBuffer []uuid.UUID

	queuedUsers, tempQueuedUsers []User
	queuedUsersMutex             sync.Mutex

	tickRate int
	ticker   *time.Ticker
}

// New constructs a new match. meta is passed to core.OnInit method.
func New(state *state.State, core Core, creator *state.User, id uuid.UUID, meta []byte) (*Match, error) {
	if id == uuid.Nil {
		id = uuid.New()
	}

	m := &Match{
		Id:       id,
		Core:     core,
		Creator:  creator,
		state:    state,
		users:    make(map[uuid.UUID]User),
		tickRate: 30,
		ticker:   time.NewTicker(time.Second / 30),
	}

	err := core.OnInit(m.State(), meta)

	return m, err
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

	for {
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
			m.users[user.User.ID()] = user
			meta, err, fatalErr := m.OnConnection(state, user, user.meta)
			user.meta = nil
			if err != nil {
				user.WritePacketTCP(knet.OCMatchJoinFail, []byte(err.Error()), nil)
			} else {
				user.WritePacketTCP(knet.OCMatchJoinSuccess, meta, user.Cipher())
			}

			if m.handleErr(fatalErr) {
				return
			}

		}

		// handle tick
		if m.handleErr(m.OnTick(state)) {
			return
		}

		<-m.ticker.C
	}
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
				user.WritePacket(opCode, data, udp, user.Cipher())
			}
		}
	} else {
		for _, user := range m.users {
			user.WritePacket(opCode, data, udp, user.Cipher())
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

// State return match state.
func (m *Match) State() State {
	return State{m.state, m}
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
