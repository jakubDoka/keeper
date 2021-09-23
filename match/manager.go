package match

import (
	"errors"
	"sync"
	"time"

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
	State        *state.State
	matches      map[uuid.UUID]*Match
	factories    map[string]func() Core
	matchesMutex sync.RWMutex
	finished     bool
}

func NewManager(state *state.State) *Manager {
	return &Manager{
		State:     state,
		matches:   make(map[uuid.UUID]*Match),
		factories: make(map[string]func() Core),
	}
}

func (m *Manager) Accept(conn *knet.Connection) {
	conn.Tcp.SetReadDeadline(time.Now().Add(1 * time.Second))
	data, disconnected, err := knet.ReadPacket(conn.Tcp)
	if disconnected || err != nil {
		conn.Close()
		return
	}
	conn.Tcp.SetReadDeadline(time.Time{})

	match, user, meta, err := m.DecodeInitialPacket(data)
	if match == nil {
		conn.WritePacketTCP(knet.OCMatchJoinFail, []byte(err.Error()), user.Cipher())
		conn.Close()
		return
	}

	go conn.CollectPackets(m.State, user.Cipher())

	match.ConnectUser(user, conn, meta)
}

func (m *Manager) DecodeInitialPacket(data []byte) (*Match, *state.User, []byte, error) {
	packet, err := knet.DecodeEncryptedClientPacket(m.State, data, false)
	if err != nil {
		return nil, nil, nil, util.WrapErr("failed to decrypt packet", err)
	}

	var matchID uuid.UUID
	if len(packet.Data) < len(matchID) {
		return nil, nil, nil, ErrMissingMatchID
	}
	copy(matchID[:], packet.Data[:])

	match := m.GetMatch(matchID)
	if match == nil {
		return nil, nil, nil, ErrMatchNotFound
	}

	return match, packet.User, packet.Data[len(matchID):], nil
}

func (m *Manager) GetMatch(id uuid.UUID) *Match {
	m.matchesMutex.RLock()
	match := m.matches[id]
	m.matchesMutex.RUnlock()
	return match
}

func (m *Manager) AddMatch(match *Match) {
	m.matchesMutex.Lock()
	m.matches[match.Id] = match
	m.matchesMutex.Unlock()

	go match.Run()
}

func (m *Manager) RegisterCore(id string, factory func() Core) {
	m.check()
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
