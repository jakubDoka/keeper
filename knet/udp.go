package knet

import (
	"net"
	"sync"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/kcrypto"
	"github.com/jakubDoka/keeper/util/uuid"
)

const UDPMaxPacketSize = 65_535

type UDPListener struct {
	conn             *net.UDPConn
	connections      map[string]*UDPPacketBuffer
	connectionsMutex sync.Mutex
	pending          map[uuid.UUID]net.Addr
	pendingMutex     sync.Mutex
}

func ListenUDP(state *state.State, addr string) (*UDPListener, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, err
	}

	listener := &UDPListener{
		conn:        conn,
		connections: make(map[string]*UDPPacketBuffer),
		pending:     make(map[uuid.UUID]net.Addr),
	}
	go listener.CollectPackets(state)
	return listener, nil
}

func (l *UDPListener) CollectPackets(state *state.State) {
	for {
		var buffer [UDPMaxPacketSize]byte
		n, addr, err := l.conn.ReadFromUDP(buffer[:])
		if err != nil {
			if util.CheckSysCallError(err, "wsarecvfrom") {
				continue
			}
			state.Fatal("udp server shut down due to error: %s", err)
		}

		packet := make([]byte, n)
		copy(packet, buffer[:n])

		str := addr.String()

		l.connectionsMutex.Lock()
		val := l.connections[str]
		l.connectionsMutex.Unlock()
		if val == nil {
			packet, err := DecodeEncryptedClientPacket(state, packet, true)
			if err != nil {
				state.Debug("Initial udp packet from %s is invalid: %s", str, err)
				continue
			}

			if packet.OpCode != OCConnectionRequest {
				state.Debug(
					"Initial udp packet from %s has invalid opcode. (%s != %s)",
					str, packet.OpCode, OCConnectionRequest,
				)
				continue
			}

			l.PutPending(packet.Session, addr)
		} else {
			val.Add(packet)
		}
	}
}

func (l *UDPListener) TakePending(session uuid.UUID) net.Addr {
	l.pendingMutex.Lock()
	val := l.pending[session]
	delete(l.pending, session)
	l.pendingMutex.Unlock()
	return val
}

func (l *UDPListener) PutPending(session uuid.UUID, val net.Addr) {
	l.pendingMutex.Lock()
	l.pending[session] = val
	l.pendingMutex.Unlock()
}

func (l *UDPListener) AddConnection(addr string) *UDPPacketBuffer {
	val := &UDPPacketBuffer{}
	l.connectionsMutex.Lock()
	l.connections[addr] = val
	l.connectionsMutex.Unlock()
	return val
}

func (l *UDPListener) RemoveConnection(addr string) {
	l.connectionsMutex.Lock()
	delete(l.connections, addr)
	l.connectionsMutex.Unlock()
}

func (l *UDPListener) WritePacket(opCode OpCode, data []byte, addr net.Addr, cipher *kcrypto.Cipher) error {
	_, err := l.conn.WriteTo(EncodePacketUDP(opCode, data, cipher), addr)
	return err
}

type UDPPacketBuffer struct {
	mutex  sync.Mutex
	buffer [][]byte
}

func (u *UDPPacketBuffer) Add(buffer []byte) {
	u.mutex.Lock()
	u.buffer = append(u.buffer, buffer)
	u.mutex.Unlock()
}

func (u *UDPPacketBuffer) HarvestPackets(buffer *[][]byte) {
	u.mutex.Lock()
	*buffer = append(*buffer, u.buffer...)
	u.buffer = u.buffer[:0]
	u.mutex.Unlock()
}
