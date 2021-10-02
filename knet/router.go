package knet

import (
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/uuid"
)

type RpcHandlerFunc = func(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error

type Router struct {
	*http.Server
	*state.State

	Listener    *Listener
	Mux         *http.ServeMux
	rpcHandlers map[string][]RpcHandlerFunc
}

func NewRouter(state *state.State) (*Router, error) {
	listener, err := NewListener(state, state.Net.GetConnectionString())
	if err != nil {
		return nil, util.WrapErr("failed to create new listener", err)
	}

	mux := http.NewServeMux()

	r := &Router{
		Listener:    listener,
		Mux:         mux,
		State:       state,
		rpcHandlers: make(map[string][]RpcHandlerFunc),
		Server:      &http.Server{},
	}

	mux.HandleFunc("/rpc", r.RpcHandler)

	return r, nil
}

func (r *Router) RegisterRpc(id string, handler ...RpcHandlerFunc) {
	r.Info("Registered rpc: %s", id)
	r.rpcHandlers[id] = handler
}

func (r *Router) Serve(addr, certFile, keyFile string) error {
	r.Server.Addr = addr
	r.Server.Handler = r.Mux

	if certFile != "" && keyFile != "" {
		return r.ListenAndServeTLS(certFile, keyFile)
	}
	r.Warn("HTTP server is running without ssl.")
	return r.ListenAndServe()
}

func (r *Router) RpcHandler(w http.ResponseWriter, re *http.Request) {
	id := re.Header.Get("id")
	if id == "" {
		http.Error(w, "rpc call needs id in headers", http.StatusBadRequest)
		return
	}

	handlers, ok := r.rpcHandlers[id]
	if !ok {
		http.Error(w, "unknown rpc id", http.StatusBadRequest)
		return
	}

	var rawSession [len(uuid.Nil) * 2]byte
	copy(rawSession[:], re.Header.Get("session"))
	var session uuid.UUID
	hex.Decode(session[:], rawSession[:])

	user := r.GetUser(session, uuid.Nil)

	r.Debug("Rpc call: id: %s session: %s", id, session)

	for _, handler := range handlers {
		err := handler(r.State, user, w, re)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

var ErrInvalidSession = errors.New("invalid session")

func RpcAssertUser(state *state.State, user *state.User, w http.ResponseWriter, re *http.Request) error {
	if user == nil {
		return ErrInvalidSession
	}

	return nil
}
