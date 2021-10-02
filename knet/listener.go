package knet

import (
	"log"
	"net"
	"time"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/kcrypto"
)

const UdpTries = 3

type Listener struct {
	*state.State

	acceptors map[string]Acceptor
	tcp       *net.TCPListener
	udp       *UDPListener
}

func NewListener(state *state.State, addr string) (*Listener, error) {
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
		State:     state,
		acceptors: make(map[string]Acceptor),
		tcp:       tcp,
		udp:       udp,
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

		l.Debug("Accepted tcp connection from %s.", conn.RemoteAddr())

		go l.Verify(conn)
	}
}

func (l *Listener) Verify(conn *net.TCPConn) {
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	data, err := ReadPacket(conn)
	if err != nil {
		l.Debug("Connection %s timed out.", conn.RemoteAddr())
		return
	}
	conn.SetReadDeadline(time.Time{})

	var cipher kcrypto.Cipher

	packet, err := DecodeEncryptedClientPacket(l.State, data, false, &cipher)
	if err != nil {
		l.Debug("Connection %s sent malformed connection request: %s", conn.RemoteAddr(), err)
		return
	}

	if packet.OpCode != OCConnectionRequest {
		l.Debug(
			"Initial packet from %s has invalid op code. (%s != %s)",
			conn.RemoteAddr(), packet.OpCode, OCConnectionRequest,
		)
		return
	}

	id := packet.User.Session()

	for i := 0; i < UdpTries; i++ {
		time.Sleep(time.Second)
		if pending := l.udp.TakePending(id); pending != nil {
			l.Accept(packet, NewConnection(conn, l.udp, pending, cipher))
			return
		}
	}

	l.Debug("%s failed to establish udp connection.", conn.RemoteAddr())
	conn.Close()
}

func (l *Listener) Accept(packet ClientPacket, conn *Connection) {
	reader := util.NewReader(packet.Data)

	acceptorID, ok := reader.String()
	if !ok {
		l.Debug("Failed to read acceptor id from %s.", conn.Tcp.RemoteAddr())
		return
	}

	acceptor, ok := l.acceptors[acceptorID]
	if !ok {
		l.Debug("Failed to find acceptor with id %s for connection %s.", acceptorID, conn.Tcp.RemoteAddr())
		return
	}

	packet.Data = reader.Rest()

	acceptor.Accept(conn, packet)

	l.DeleteKey(packet.User.ID())
}

func (l *Listener) RegisterAcceptor(id string, acceptor Acceptor) {
	l.Info("Registering acceptor under %s.", id)
	l.acceptors[id] = acceptor
}

type Acceptor interface {
	Accept(*Connection, ClientPacket)
}
