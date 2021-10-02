package state

import (
	"database/sql"
	"sync"
	"time"

	"github.com/jakubDoka/keeper/kcfg"
	"github.com/jakubDoka/keeper/klog"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/kcrypto"
	"github.com/jakubDoka/keeper/util/uuid"
)

// State holds application state. All allowed operations on state are thread safe.
type State struct {
	*kcfg.Config
	*sql.DB
	*klog.Logger
	*util.Prepared

	sessions     map[uuid.UUID]*User
	users        map[uuid.UUID]*User
	sessionMutex sync.RWMutex

	keys     map[uuid.UUID]kcrypto.Key
	keyMutex sync.RWMutex
}

// New creates new state.
func New(db *sql.DB, cfg *kcfg.Config, log *klog.Logger) *State {
	state := &State{
		Config:   cfg,
		DB:       db,
		Logger:   log,
		Prepared: util.NewPrepared(),
		sessions: make(map[uuid.UUID]*User),
		users:    make(map[uuid.UUID]*User),
		keys:     make(map[uuid.UUID]kcrypto.Key),
	}
	return state
}

func (s *State) Prepare(id, content string) error {
	return s.Prepared.Prepare(s.DB, id, content)
}

func (s *State) CreateKey(userID uuid.UUID) kcrypto.Key {
	key := kcrypto.NewKey()

	s.keyMutex.Lock()
	s.keys[userID] = key
	s.keyMutex.Unlock()

	return key
}

func (s *State) GetKey(userID uuid.UUID) (key kcrypto.Key, ok bool) {
	s.keyMutex.RLock()
	key, ok = s.keys[userID]
	s.keyMutex.RUnlock()
	return
}

func (s *State) DeleteKey(userID uuid.UUID) {
	s.keyMutex.Lock()
	delete(s.keys, userID)
	s.keyMutex.Unlock()
}

// AddUser adds user to state so it is accessable. Session is access point that you should
// send to user so he can verify himself.
func (s *State) AddUser(user *User) {
	s.sessionMutex.Lock()
	s.sessions[user.session] = user
	if user, ok := s.users[user.id]; ok {
		delete(s.users, user.id)
	}
	s.users[user.id] = user
	s.sessionMutex.Unlock()
}

// GetUser returns user under the session if present. If user expired or does not exist,
// nil is returned.
func (s *State) GetUser(session, id uuid.UUID) *User {
	var user *User
	var ok bool

	s.sessionMutex.RLock()
	if session == uuid.Nil {
		user, ok = s.users[id]
	} else {
		user, ok = s.sessions[session]
	}
	s.sessionMutex.RUnlock()

	if !ok {
		return nil
	}

	if user.Expired() {
		s.sessionMutex.Lock()
		delete(s.sessions, user.session)
		delete(s.users, user.id)
		s.sessionMutex.Unlock()

		s.DeleteKey(user.id)

		return nil
	}

	return user
}

// User holds minimal data about user that is required by system.
// All allowed operations on user are thread safe.
type User struct {
	id, session uuid.UUID
	expiration  time.Time
	ip          string
}

// NewUser constructs a user with given livetime. User is also give a cipher
// to encrypt and decrypt his messages.
func NewUser(id, session uuid.UUID, duration time.Duration, IP string) *User {
	return &User{
		id:         id,
		session:    session,
		expiration: time.Now().Add(duration),
		ip:         IP,
	}
}

// ID return user id.
func (u *User) ID() uuid.UUID {
	return u.id
}

// IP returns user ip.
func (u *User) IP() string {
	return u.ip
}

// Session returns user session.
func (u *User) Session() uuid.UUID {
	return u.session
}

func (u *User) Expired() bool {
	return u.expiration.Before(time.Now())
}

func (u *User) Expiration() time.Time {
	return u.expiration
}
