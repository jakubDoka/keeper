package core

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"reflect"

	"github.com/jakubDoka/keeper/kcfg"
	"github.com/jakubDoka/keeper/klog"
	"github.com/jakubDoka/keeper/knet"
	"github.com/jakubDoka/keeper/match"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/uuid"
	_ "github.com/lib/pq"
)

var ConfigPath = flag.String("config", "kconfig.yaml", "Path to config file.")

// Launch calls LaunchWithConfig with default config.
func Launch(mods ...Module) App {
	return LaunchWithConfig(nil, mods...)
}

// LaunchWithConfig launches the application accepting modules on which the Init will be called.
func LaunchWithConfig(config *kcfg.Config, mods ...Module) App {
	flag.Parse()

	if config == nil {
		c, err := kcfg.Load(*ConfigPath)
		if err != nil {
			fmt.Println("failed to load config, (using default):", err)
		}
		config = &c
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
	router, err := knet.NewRouter(s)
	if err != nil {
		logger.Fatal("cannot create router: %s", err)
	}

	router.Listener.RegisterAcceptor("match", matchManager)

	app := App{s, router, matchManager}
	app.createMatchHandler()
	app.createKeyHandler()

	if len(mods) > 0 {
		for _, mod := range mods {
			logger.Info("Loading %s module...", reflect.TypeOf(mod).Name())
			mod.Init(app)
		}
	}

	logger.Finish()
	s.Prepared.Finish()

	logger.Info("Starting HTTP server (%s)...", config.Net.GetHttpConnectionString())
	go func() {
		err := router.Serve(config.Net.GetHttpConnectionString(), config.Net.CertFile, config.Net.KeyFile)
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Http server shut down due to error: %s", err)
		}
	}()

	return app
}

type App struct {
	*state.State
	*knet.Router
	*match.Manager
}

type Module interface {
	Init(a App)
}

var (
	ErrMissingPassword = errors.New("missing password")
	ErrMissingEmail    = errors.New("missing email")
)

func (a App) RegisterEmailRegisterHandler(handler func(state *state.State, email, password string, meta []byte) error) {
	a.RegisterRpc("register-email", func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
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

func (a App) RegisterEmailLoginHandler(handler func(state *state.State, email, password, addr string) (*state.User, error)) {
	a.RegisterRpc("login-email", func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
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
		writer := calc.
			UUID().
			UUID().
			String(re.RemoteAddr).
			Uint64().
			ToWriter()
		writer.
			UUID(user.ID()).
			UUID(user.Session()).
			String(re.RemoteAddr).
			Uint64(uint64(user.Expiration().Unix()))

		fmt.Println(uint64(user.Expiration().Unix()))

		w.Write(writer.Buffer())

		state.AddUser(user)

		return nil
	})
}

func (a App) createMatchHandler() {
	a.RegisterRpc("create-match", knet.RpcAssertUser, func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
		reader, err := util.BodyToReader(re)
		if err != nil {
			return err
		}

		factoryID, ok := reader.String()
		if !ok {
			return errors.New("missing match type")
		}

		factory := a.GetCore(factoryID)
		if factory == nil {
			return errors.New("unknown match type")
		}

		core := factory()
		matchID := uuid.New()
		match, err := match.New(state, a.Manager, core, user, matchID, reader.Rest())
		if err != nil {
			return err
		}

		a.AddMatch(match)

		w.Write(matchID[:])

		return nil
	})
}

func (a App) createKeyHandler() {
	a.RegisterRpc("create-key", knet.RpcAssertUser, func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
		if _, ok := state.GetKey(user.ID()); ok {
			return errors.New("you already have key so use it, then you can ask for more")
		}

		key := state.CreateKey(user.ID())

		w.Write(key[:])

		return nil
	})
}

func Block() {
	var ch chan struct{}
	<-ch
}
