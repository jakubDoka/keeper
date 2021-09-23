package knet

import (
	"log"
	"net"
	"time"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
)

const UdpTries = 3

type Acceptor interface {
	Accept(*Connection)
}

type Listener struct {
	acceptor Acceptor
	tcp      *net.TCPListener
	udp      *UDPListener
	state    *state.State
}

func NewListener(state *state.State, addr string, acceptor Acceptor) (*Listener, error) {
	address, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, util.WrapErr("failed to resolve TCP address", err)
	}

	state.Info("Listening TCP (%s)...", addr)
	tcp, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, util.WrapErr("failed to create inner TCP listener", err)
	}

	state.Info("Listening UDP (%s)...", addr)
	udp, err := ListenUDP(state, addr)
	if err != nil {
		return nil, util.WrapErr("failed to create inner UDP listener", err)
	}

	result := &Listener{
		acceptor: acceptor,
		tcp:      tcp,
		udp:      udp,
		state:    state,
	}

	go result.Run()

	return result, nil
}

func (l *Listener) Run() {
	for {
		conn, err := l.tcp.AcceptTCP()
		if err != nil {
			log.Fatal("tcp server shut down due to error: ", err)
			return
		}

		l.state.Debug("Accepted tcp connection from %s.", conn.RemoteAddr())

		go l.Verify(conn)
	}
}

func (l *Listener) Verify(conn *net.TCPConn) {
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	data, disconnected, err := ReadPacket(conn)
	if disconnected || err != nil {
		l.state.Debug("Connection %s timed out.", conn.RemoteAddr())
		return
	}
	conn.SetReadDeadline(time.Time{})

	packet, err := DecodeEncryptedClientPacket(l.state, data, false)
	if err != nil {
		l.state.Debug("Connection %s sent malformed connection request: %s", conn.RemoteAddr(), err)
		return
	}

	if packet.OpCode != OCConnectionRequest {
		l.state.Debug(
			"Initial packet from %s has invalid op code. (%s != %s)",
			conn.RemoteAddr(), packet.OpCode, OCConnectionRequest,
		)
		return
	}

	id := packet.User.Session()

	for i := 0; i < UdpTries; i++ {
		time.Sleep(time.Second)
		if pending := l.udp.TakePending(id); pending != nil {
			l.acceptor.Accept(NewConnection(conn, l.udp, pending))
			return
		}
	}

	l.state.Debug("%s failed to establish udp connection.", conn.RemoteAddr())
	conn.Close()
}
