package knet

import (
	"crypto/aes"
	"encoding/binary"
	"errors"
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
	ErrMissingUserID      = errors.New("packet is missing user id")
	ErrMissingGen         = errors.New("packet is missing gen")
	ErrMissingKey         = errors.New("there is no key for initial packet")
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

	cipher.EncryptTCP(writer.Buffer()[4:]) // skip size
	return writer.Buffer()[:calc.Value()]  // reveal the padding
}

func EncodePacketUDP(packetCode OpCode, packetData []byte, cipher *kcrypto.Cipher) []byte {
	var calc util.Calculator
	writer := calc.
		Uint32().
		Rest(packetData).
		Pad(aes.BlockSize).
		Uint32().
		ToWriter()
	writer.
		Uint32(0).
		Uint32(uint32(packetCode)).
		Rest(packetData)

	_, gen := cipher.EncryptUDP(writer.Buffer()[4:])
	binary.BigEndian.PutUint32(writer.Buffer(), gen)
	return writer.Buffer()[:calc.Value()]
}

func DecodeEncryptedClientPacket(state *state.State, data []byte, udp bool, cipher *kcrypto.Cipher) (ClientPacket, error) {
	reader := util.NewReader(data)

	id, ok := reader.UUID()
	if !ok {
		return ClientPacket{}, ErrMissingUserID
	}

	if cipher.IsNil() {
		key, ok := state.GetKey(id)
		if !ok {
			return ClientPacket{}, ErrMissingKey
		}
		*cipher = kcrypto.NewCipherWithKey(key)
	}

	var err error
	if udp {
		gen, ok := reader.Uint32()
		if !ok {
			return ClientPacket{}, ErrMissingGen
		}
		data, err = cipher.DecryptUDP(reader.Rest(), gen)
	} else {
		data, err = cipher.DecryptTCP(reader.Rest())
	}
	if err != nil {
		return ClientPacket{}, util.WrapErr("failed to decrypt packet", err)
	}

	packet, err := DecodeClientPacket(data, udp)
	if err != nil {
		return ClientPacket{}, util.WrapErr("failed to decode packet", err)
	}

	user := state.GetUser(uuid.Nil, id)

	if user == nil {
		return ClientPacket{}, ErrIDOrSessionInvalid
	}

	if packet.Session != user.Session() {
		return packet, ErrSessionInvalid
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

func ReadPacket(conn net.Conn) ([]byte, error) {
	var packetSize [4]byte
	_, err := conn.Read(packetSize[:])
	if err != nil {
		return nil, util.WrapErr("failed to read packet size", err)
	}

	size := binary.BigEndian.Uint32(packetSize[:])
	buffer := make([]byte, size)

	_, err = conn.Read(buffer)
	if err != nil {
		return nil, util.WrapErr("failed to read packet content", err)
	}

	return buffer, nil
}
