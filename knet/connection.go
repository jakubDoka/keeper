package knet

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util/kcrypto"
)

type Connection struct {
	Tcp *net.TCPConn
	Udp *UDPListener

	UdpAddr net.Addr
	UdpBuff *UDPPacketBuffer

	cipher kcrypto.Cipher

	disconnected int32

	queuedPackets      []ClientPacket
	queuedPacketsMutex sync.Mutex
}

func NewConnection(tcp *net.TCPConn, udp *UDPListener, udpAddr net.Addr, cipher kcrypto.Cipher) *Connection {
	return &Connection{
		Tcp:     tcp,
		Udp:     udp,
		UdpAddr: udpAddr,
		UdpBuff: udp.AddConnection(udpAddr.String()),
		cipher:  cipher,
	}
}

// run this on thread
func (c *Connection) CollectPackets(state *state.State) {
	for {
		data, err := ReadPacket(c.Tcp)
		if err != nil {
			state.Debug("Connection %s disconnected due to error: %s", c.Tcp.RemoteAddr(), err)
			c.markDisconnected()
			return
		}

		packet, err := DecodeEncryptedClientPacket(state, data, false, &c.cipher)
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
		packet, err := DecodeEncryptedClientPacket(state, data, true, &c.cipher)
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

func (c *Connection) WritePacket(packetCode OpCode, packetData []byte, udp bool) error {
	if udp {
		return c.WritePacketUDP(packetCode, packetData)
	}
	return c.WritePacketTCP(packetCode, packetData)
}

func (c *Connection) WritePacketTCP(packetCode OpCode, packetData []byte) error {
	_, err := c.Tcp.Write(EncodePacketTCP(packetCode, packetData, &c.cipher))
	return err
}

func (c *Connection) WritePacketUDP(packetCode OpCode, packetData []byte) error {
	_, err := c.Udp.conn.WriteTo(EncodePacketUDP(packetCode, packetData, &c.cipher), c.UdpAddr)
	return err
}

func (c *Connection) markDisconnected() {
	atomic.StoreInt32(&c.disconnected, 1)
}

func (c *Connection) Disconnected() bool {
	return atomic.LoadInt32(&c.disconnected) == 1
}

func (c *Connection) Cipher() *kcrypto.Cipher {
	return &c.cipher
}

func (c *Connection) Close() {
	c.Tcp.Close()
	c.Udp.RemoveConnection(c.UdpAddr.String())
}
