package core

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"reflect"

	"github.com/jakubDoka/keeper/cfg"
	"github.com/jakubDoka/keeper/klog"
	"github.com/jakubDoka/keeper/knet"
	"github.com/jakubDoka/keeper/match"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/uuid"
	_ "github.com/lib/pq"
)

var ConfigPath = flag.String("config", "config.yaml", "Path to config file.")

// Launch launches the application accepting modules on which the Init will be called.
func Launch(mods ...Module) {
	flag.Parse()

	config, err := cfg.Load(*ConfigPath)
	if err != nil {
		fmt.Println("failed to load config, (using default):", err)
	}

	logger := &klog.Logger{}

	logger.ApplyConfig(config.Log)

	logger.Info("Connecting database...")
	db, err := sql.Open(config.Db.Driver, config.Db.GetConnectionString())
	if err == nil {
		err = db.Ping()
	}
	if err != nil {
		logger.Fatal("cannot connect to database: %s", err)
	}
	if config.Db.SSLMode == "disable" {
		logger.Warn("Database is running without ssl.")
	}

	s := state.New(db, config, logger)

	matchManager := match.NewManager(s)

	logger.Info("Initializing router...")
	router, err := knet.NewRouter(s, matchManager)
	if err != nil {
		logger.Fatal("cannot create router: %s", err)
	}

	export := InitState{s, router, matchManager}
	export.createMatchHandler()

	if len(mods) > 0 {
		for _, mod := range mods {
			logger.Info("Loading %s module...", reflect.TypeOf(mod).Name())
			mod.Init(export)
		}
	}

	logger.Finish()
	s.Prepared.Finish()

	logger.Info("Starting HTTP server (%s)...", config.Net.GetHttpConnectionString())
	err = router.Serve(config.Net.CertFile, config.Net.KeyFile)
	if err != nil {
		logger.Fatal("Http server shut down due to error: %s", err)
	}
}

type Module interface {
	Init(export InitState)
}

type InitState struct {
	*state.State
	*knet.Router
	*match.Manager
}

var (
	ErrMissingPassword = errors.New("missing password")
	ErrMissingEmail    = errors.New("missing email")
)

func (m InitState) RegisterEmailRegisterHandler(handler func(state *state.State, email, password string, meta []byte) error) {
	m.RegisterRpc("register-email", func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
		reader, err := util.BodyToReader(re)
		if err != nil {
			return err
		}

		email, ok := reader.String()
		if !ok {
			return ErrMissingEmail
		}

		password, ok := reader.String()
		if !ok {
			return ErrMissingPassword
		}

		meta := reader.Rest()

		err = handler(state, email, password, meta)
		if err != nil {
			return err
		}

		w.Write([]byte("OK"))

		return nil
	})
}

func (m InitState) RegisterEmailLoginHandler(handler func(state *state.State, email, password, addr string) (*state.User, error)) {
	m.RegisterRpc("login-email", func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
		reader, err := util.BodyToReader(re)
		if err != nil {
			return err
		}

		email, ok := reader.String()
		if !ok {
			return ErrMissingEmail
		}

		password, ok := reader.String()
		if !ok {
			return ErrMissingPassword
		}

		user, err = handler(state, email, password, re.RemoteAddr)
		if err != nil {
			return err
		}

		var calc util.Calculator
		writer := calc.UUID().UUID().String(re.RemoteAddr).Key().ToWriter()
		writer.
			UUID(user.ID()).
			UUID(user.Session()).
			String(re.RemoteAddr).
			Key(user.Cipher().Key())

		w.Write(writer.Buffer())

		state.AddUser(user)

		return nil
	})
}

func (m InitState) createMatchHandler() {
	m.RegisterRpc("create-match", knet.RpcAssertUser, func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
		reader, err := util.BodyToReader(re)
		if err != nil {
			return err
		}

		factoryID, ok := reader.String()
		if !ok {
			return errors.New("missing match type")
		}

		factory := m.GetCore(factoryID)
		if factory == nil {
			return errors.New("unknown match type")
		}

		core := factory()
		matchID := uuid.New()
		match, err := match.New(state, core, user, matchID, reader.Rest())
		if err != nil {
			return err
		}

		m.AddMatch(match)

		w.Write(matchID[:])

		return nil
	})
}
