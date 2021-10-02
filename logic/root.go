package logic

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/jakubDoka/keeper/core"
	"github.com/jakubDoka/keeper/logic/auth"
	"github.com/jakubDoka/keeper/logic/cfg"
	"github.com/jakubDoka/keeper/logic/pages"
	"github.com/jakubDoka/keeper/logic/users"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/uuid"
)

var (
	ErrFailedToSendEmail  = errors.New("Failed to send email to you, the email address may not exist.")
	ErrInvalidLogin       = errors.New("Password or email is incorrect.")
	ErrMissingMax         = errors.New("Missing max match amount.")
	ErrMissingMin         = errors.New("Missing min match amount.")
	ErrMissingMatchRation = errors.New("Missing match ratio.")
)

func Run() {
	core.Launch(&Mod{})

	core.Block()
}

type Mod struct {
	core.App
	*cfg.Config

	EmailClient auth.EmailClient

	PendigUsers      map[uuid.UUID]users.Unverified
	PendigUsersMutex sync.Mutex
}

func (m *Mod) Init(a core.App) {
	confug, err := cfg.Load("config.yaml")
	if err != nil {
		a.Fatal(err.Error())
	}

	m.App = a
	m.Config = confug
	m.EmailClient = auth.NewEmailClient(confug.Email.Value, confug.Email.Password)
	m.PendigUsers = make(map[uuid.UUID]users.Unverified)

	a.Mux.Handle("/static/", http.FileServer(http.FS(pages.Static)))

	a.RegisterEmailRegisterHandler(m.HandleRegistration)
	a.RegisterEmailLoginHandler(m.HandleLogin)
	a.RegisterRpc("find-match", m.SearchMatch)
	a.Mux.HandleFunc("/verify", m.VerifyUser)

}

func (m *Mod) HandleLogin(s *state.State, email string, password string, addr string) (*state.User, error) {
	id, ok := users.Verify(s, email, password)
	if !ok {
		return nil, ErrInvalidLogin
	}

	user := state.NewUser(id, uuid.New(), time.Duration(m.SessionDuration)*time.Minute, addr)

	return user, nil
}

func (m *Mod) HandleRegistration(state *state.State, email, password string, meta []byte) error {
	user, err := users.NewUnverified(email, password)
	if err != nil {
		return err
	}

	url := url.URL{
		Host: state.Net.GetHttpConnectionString(),
		Path: "/verify",
		RawQuery: url.Values{
			"code": {user.Code.String()},
		}.Encode(),
	}

	err = m.EmailClient.Send(url.String(), user.Email)
	if err != nil {
		state.Error(err.Error())
		return ErrFailedToSendEmail
	}

	m.PendigUsersMutex.Lock()
	m.PendigUsers[user.Code] = user
	m.PendigUsersMutex.Unlock()
	return nil
}

func (m *Mod) VerifyUser(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rawCode := query.Get("code")
	if rawCode == "" {
		http.Error(w, "Missing code.", http.StatusBadRequest)
		return
	}

	code, err := uuid.Parse(rawCode)
	if err != nil {
		http.Error(w, "Invalid code.", http.StatusBadRequest)
		return
	}

	m.PendigUsersMutex.Lock()
	user, ok := m.PendigUsers[code]
	if ok {
		delete(m.PendigUsers, code)
	}
	m.PendigUsersMutex.Unlock()

	if !ok {
		http.Error(w, "Invalid code.", http.StatusBadRequest)
		return
	}

	if time.Now().After(user.Expiration) {
		http.Error(w, "Code expired.", http.StatusBadRequest)
		return
	}

	_, err = users.Create(m.State, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Write([]byte("Your account was created."))
}

func (m *Mod) SearchMatch(state *state.State, user *state.User, w http.ResponseWriter, r *http.Request) error {
	reader, err := util.BodyToReader(r)
	if err != nil {
		return err
	}

	max, ok := reader.Uint32()
	if !ok {
		return ErrMissingMax
	}

	matchRatio, ok := reader.Uint32()
	if !ok {
		return ErrMissingMatchRation
	}

	query := reader.Rest()

	matches, err := m.Manager.Search(max, matchRatio, query)
	if err != nil {
		return err
	}

	writer := util.NewWriter(len(matches) * 0xFF)
	writer.Uint32(uint32(len(matches)))

	for _, match := range matches {
		writer.UUID(match)
		match := m.Manager.GetMatch(match)
		writer.Uint32(match.UserAmount())
		writer.Bytes(match.Info())
	}

	w.Write(writer.Buffer())

	return nil
}
