package match

import (
	"errors"
	"fmt"
	"sync"

	"github.com/jakubDoka/keeper/index"
	"github.com/jakubDoka/keeper/knet"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/uuid"
)

var (
	ErrMissingMatchID = errors.New("missing match id")
	ErrMatchNotFound  = errors.New("match not found")
)

type Manager struct {
	*state.State

	index        *index.Index
	matches      map[uuid.UUID]*Match
	factories    map[string]func() Core
	matchesMutex sync.RWMutex
	finished     bool
}

func NewManager(state *state.State) *Manager {
	return &Manager{
		State:     state,
		index:     index.New(),
		matches:   make(map[uuid.UUID]*Match),
		factories: make(map[string]func() Core),
	}
}

func (m *Manager) Search(max, ratio uint32, query []byte) ([]uuid.UUID, error) {
	max = util.Clamp(max, 1, 100)
	result := make([]uuid.UUID, 0, max)

	if len(query) == 0 {
		m.matchesMutex.Lock()
		for id := range m.matches {
			result = append(result, id)
		}
		m.matchesMutex.Unlock()
		return result, nil
	}

	buffer := Buffer{}

	var parser index.Parser

	fields, i, err := parser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query:%d: %s", i, err)
	}

	m.index.Search(&buffer, fields...)

	for id, count := range buffer {
		if count >= ratio {
			result = append(result, id)
		}
		if uint32(len(result)) >= max {
			break
		}
	}

	return result, nil
}

func (m *Manager) Accept(conn *knet.Connection, packet knet.ClientPacket) {
	reader := util.NewReader(packet.Data)

	matchID, ok := reader.UUID()
	if !ok {
		m.Debug("Packet from %s is missing match id.", conn.Tcp.RemoteAddr())
		return
	}

	match := m.GetMatch(matchID)
	if match == nil {
		conn.WritePacketTCP(knet.OCMatchJoinFail, []byte("Match with this id does not exist."))
		return
	}

	go conn.CollectPackets(m.State)

	match.ConnectUser(packet.User, conn, reader.Rest())
}

func (m *Manager) GetMatch(id uuid.UUID) *Match {
	m.matchesMutex.RLock()
	match := m.matches[id]
	m.matchesMutex.RUnlock()
	return match
}

func (m *Manager) AddMatch(match *Match) {
	m.matchesMutex.Lock()
	m.matches[match.id] = match
	m.matchesMutex.Unlock()

	go match.Run()
}

func (m *Manager) RegisterCore(id string, factory func() Core) {
	m.check()
	m.Info("Registered match core under %s.", id)
	m.factories[id] = factory
}

func (m *Manager) GetCore(id string) func() Core {
	return m.factories[id]
}

func (m *Manager) check() {
	if m.finished {
		panic("match manager already finished, do this during initialization")
	}
}

func (m *Manager) Finish() {
	m.finished = true
}

func (m *Manager) Finished() bool {
	return m.finished
}

type Buffer map[uuid.UUID]uint32

func (b *Buffer) Add(id interface{}) {
	(*b)[*id.(*uuid.UUID)]++
}
