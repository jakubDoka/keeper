package knet

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util/kcrypto"
)

type Connection struct {
	Tcp                *net.TCPConn
	Udp                *UDPListener
	UdpAddr            net.Addr
	UdpBuff            *UDPPacketBuffer
	disconnected       int32
	queuedPackets      []ClientPacket
	queuedPacketsMutex sync.Mutex
}

func NewConnection(tcp *net.TCPConn, udp *UDPListener, udpAddr net.Addr) *Connection {
	return &Connection{
		Tcp:     tcp,
		Udp:     udp,
		UdpAddr: udpAddr,
		UdpBuff: udp.AddConnection(udpAddr.String()),
	}
}

// run this on thread
func (c *Connection) CollectPackets(state *state.State, cipher *kcrypto.Cipher) {
	for {
		data, disconnected, err := ReadPacket(c.Tcp)
		if disconnected {
			state.Debug("Connection %s disconnected.", c.Tcp.RemoteAddr())
			c.markDisconnected()
			return
		}

		if err != nil {
			state.Debug("Error when reading connection: %s", err)
			continue
		}

		packet, err := DecodeEncryptedClientPacket(state, data, false)
		if err != nil {
			state.Debug("Error when decoding packet from %s: %s", c.Tcp.RemoteAddr(), err)
			continue
		}

		c.queuedPacketsMutex.Lock()
		c.queuedPackets = append(c.queuedPackets, packet)
		c.queuedPacketsMutex.Unlock()
	}
}

func (c *Connection) HarvestPackets(state *state.State, buffer *[]ClientPacket, helper *[][]byte) {
	c.UdpBuff.HarvestPackets(helper)

	for _, data := range *helper {
		packet, err := DecodeEncryptedClientPacket(state, data, true)
		if err != nil {
			state.Debug("Error when decoding packet from %s: %s", c.Tcp.RemoteAddr(), err)
			continue
		}
		*buffer = append(*buffer, packet)
	}

	c.queuedPacketsMutex.Lock()
	*buffer = append(*buffer, c.queuedPackets...)
	c.queuedPackets = c.queuedPackets[:0]
	c.queuedPacketsMutex.Unlock()
}

func (c *Connection) WritePacket(packetCode OpCode, packetData []byte, udp bool, cipher *kcrypto.Cipher) error {
	if udp {
		return c.WritePacketUDP(packetCode, packetData, cipher)
	}
	return c.WritePacketTCP(packetCode, packetData, cipher)
}

func (c *Connection) WritePacketTCP(packetCode OpCode, packetData []byte, cipher *kcrypto.Cipher) error {
	_, err := c.Tcp.Write(EncodePacketTCP(packetCode, packetData, cipher))
	return err
}

func (c *Connection) WritePacketUDP(packetCode OpCode, packetData []byte, cipher *kcrypto.Cipher) error {
	_, err := c.Udp.conn.WriteTo(EncodePacketTCP(packetCode, packetData, cipher), c.UdpAddr)
	return err
}

func (c *Connection) markDisconnected() {
	atomic.StoreInt32(&c.disconnected, 1)
}

func (c *Connection) Disconnected() bool {
	return atomic.LoadInt32(&c.disconnected) == 1
}

func (c *Connection) Close() {
	c.Tcp.Close()
	c.Udp.RemoveConnection(c.UdpAddr.String())
}
