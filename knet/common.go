package knet

import (
	"crypto/aes"
	"encoding/binary"
	"errors"
	"io"
	"net"

	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/kcrypto"
	"github.com/jakubDoka/keeper/util/uuid"
)

type OpCode uint32

const (
	OCError OpCode = iota
	OCConnectionRequest
	OCMatchJoinFail
	OCMatchJoinSuccess

	OCLast
)

var opCodeStrings = [...]string{
	"Error",
	"ConnectionRequest",
	"MatchJoinFail",
	"MatchJoinSuccess",
}

func (o OpCode) String() string {
	if o.Custom() {
		return "Custom"
	}
	return opCodeStrings[o]
}

func (o OpCode) Custom() bool {
	return o >= OCLast
}

// errors
var (
	ErrMissingPacketSize  = errors.New("expected 4 bytes as encoded packet size")
	ErrMissingSession     = errors.New("packet is missing session")
	ErrMissingCode        = errors.New("packet is missing code")
	ErrMissingTargetCount = errors.New("packet is missing target count")
	ErrMissingTarget      = errors.New("packet is missing target")
	ErrIDOrSessionInvalid = errors.New("user id or session is invalid")
	ErrSessionInvalid     = errors.New("session is invalid")
)

type ClientPacket struct {
	OpCode  OpCode
	Session uuid.UUID
	Targets []uuid.UUID
	Data    []byte
	Udp     bool
	User    *state.User
}

func EncodePacket(packetCode OpCode, packetData []byte, udp bool, cipher *kcrypto.Cipher) []byte {
	if udp {
		return EncodePacketUDP(packetCode, packetData, cipher)
	}
	return EncodePacketTCP(packetCode, packetData, cipher)
}

func EncodePacketTCP(packetCode OpCode, packetData []byte, cipher *kcrypto.Cipher) []byte {
	var calc util.Calculator
	writer := calc.
		Uint32().
		Rest(packetData).
		Pad(aes.BlockSize). // this guarate that cipher Encrypt will not reallocate
		Uint32().
		ToWriter()
	writer.
		Uint32(uint32(calc.Value() - 4)).
		Uint32(uint32(packetCode)).
		Rest(packetData)

	cipher.Encrypt(writer.Buffer()[4:], true) // skip size

	return writer.Buffer()[:calc.Value()] // reveal the padding
}

func EncodePacketUDP(packetCode OpCode, packetData []byte, cipher *kcrypto.Cipher) []byte {
	var calc util.Calculator
	writer := calc.Uint32().Rest(packetData).ToWriter()
	writer.
		Uint32(uint32(packetCode)).
		Rest(packetData)

	return cipher.Encrypt(writer.Buffer(), false)
}

func DecodeEncryptedClientPacket(state *state.State, data []byte, udp bool) (ClientPacket, error) {
	var id uuid.UUID
	copy(id[:], data)

	user := state.GetUser(uuid.Nil, id)

	if user == nil {
		return ClientPacket{}, ErrIDOrSessionInvalid
	}

	data, err := user.Cipher().Decrypt(data[len(id):], !udp)
	if err != nil {
		return ClientPacket{}, util.WrapErr("failed to decrypt packet", err)
	}

	packet, err := DecodeClientPacket(data, udp)
	if err != nil {
		return ClientPacket{}, util.WrapErr("failed to decode packet", err)
	}

	if packet.Session != user.Session() {
		return ClientPacket{}, ErrSessionInvalid
	}

	packet.User = user

	return packet, nil
}

func DecodeClientPacket(data []byte, udp bool) (ClientPacket, error) {
	reader := util.NewReader(data)

	session, ok := reader.UUID()
	if !ok {
		return ClientPacket{}, ErrMissingSession
	}

	opCode, ok := reader.Uint32()
	if !ok {
		return ClientPacket{}, ErrMissingCode
	}

	targetCount, ok := reader.Uint32()
	if !ok {
		return ClientPacket{}, ErrMissingTargetCount
	}

	targets := make([]uuid.UUID, targetCount)
	for i := range targets {
		target, ok := reader.UUID()
		if !ok {
			return ClientPacket{}, ErrMissingTarget
		}
		targets[i] = target
	}

	return ClientPacket{
		OpCode:  OpCode(opCode),
		Session: session,
		Targets: targets,
		Data:    reader.Rest(),
		Udp:     udp,
	}, nil
}

func ReadPacket(conn net.Conn) ([]byte, bool, error) {
	var packetSize [4]byte
	_, err := conn.Read(packetSize[:])
	if err != nil {
		if err == io.EOF {
			return nil, true, nil
		}
		return nil, false, util.WrapErr("failed to read packet size", err)
	}

	size := binary.BigEndian.Uint32(packetSize[:])
	buffer := make([]byte, size)

	_, err = conn.Read(buffer)
	if err != nil {
		if err == io.EOF {
			return nil, true, nil
		}
		return nil, false, util.WrapErr("failed to read packet content", err)
	}

	return buffer, false, nil
}
