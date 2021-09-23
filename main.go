package main

import (
	"time"

	"github.com/jakubDoka/keeper/core"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util/uuid"
)

func main() {
	core.Launch(MyFirstModule{})
}

type MyFirstModule struct{}

func (MyFirstModule) Init(i core.InitState) {
	i.Logger.Info("Hello, World!")

	i.RegisterEmailLoginHandler(func(s *state.State, email, password, addr string) (*state.User, error) {
		s.Debug("Email login handler called with email %s and password %s from address %s", email, password, addr)
		return state.NewUser(uuid.New(), uuid.New(), 10*time.Second, addr), nil
	})
}
