package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// Packet types
const (
	TypeMouseMove  uint8 = 1 // Mouse movement (Value: dx, Extra: dy)
	TypeMouseBtn   uint8 = 2 // Mouse button (Code: button, Value: 1=down/0=up)
	TypeMouseWheel uint8 = 3 // Mouse wheel (Value: v_wheel, Extra: h_wheel)
	TypeKey        uint8 = 4 // Keyboard event (Code: scancode, Value: 1=down/0=up/2=repeat)
	TypeRelease    uint8 = 5 // Release control back to host
	TypeClipboard  uint8 = 6 // Shared clipboard data (Data: UTF-8 text payload)
	TypeHeartbeat  uint8 = 7 // KeepAlive ping
	TypeHandshake  uint8 = 8 // Server sends layout position to client (Data: position string)
)

// Mouse button identifiers
const (
	BtnLeft   uint16 = 1
	BtnRight  uint16 = 2
	BtnMiddle uint16 = 3
	BtnSide   uint16 = 4 // Back button
	BtnExtra  uint16 = 5 // Forward button
)

// HeaderSize is the fixed size in bytes for the packet header (16 bytes)
const HeaderSize = 16

// Packet represents a structured BorderPass event packet
type Packet struct {
	Type  uint8
	Flags uint8
	Code  uint16
	Value int32
	Extra int32
	Data  []byte // Used for variable-length payloads like TypeClipboard
}

// EncodeHeader serializes the 16-byte fixed header
func (p *Packet) EncodeHeader() [HeaderSize]byte {
	var buf [HeaderSize]byte
	buf[0] = p.Type
	buf[1] = p.Flags
	binary.LittleEndian.PutUint16(buf[2:4], p.Code)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(p.Value))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(p.Extra))
	dataLen := uint32(len(p.Data))
	binary.LittleEndian.PutUint32(buf[12:16], dataLen)
	return buf
}

// SendPacket transmits a packet over a network connection
func SendPacket(conn net.Conn, p Packet) error {
	header := p.EncodeHeader()
	if len(p.Data) == 0 {
		_, err := conn.Write(header[:])
		return err
	}

	// For packets with payload (e.g. clipboard text)
	fullBuf := make([]byte, HeaderSize+len(p.Data))
	copy(fullBuf[:HeaderSize], header[:])
	copy(fullBuf[HeaderSize:], p.Data)
	_, err := conn.Write(fullBuf)
	return err
}

// ReadPacket reads a single complete packet from the stream
func ReadPacket(r io.Reader) (Packet, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Packet{}, err
	}

	pkt := Packet{
		Type:  header[0],
		Flags: header[1],
		Code:  binary.LittleEndian.Uint16(header[2:4]),
		Value: int32(binary.LittleEndian.Uint32(header[4:8])),
		Extra: int32(binary.LittleEndian.Uint32(header[8:12])),
	}

	dataLen := binary.LittleEndian.Uint32(header[12:16])
	if dataLen > 0 {
		// Prevent memory exhaustion attacks: max 10MB payload
		if dataLen > 10*1024*1024 {
			return Packet{}, fmt.Errorf("payload size too large: %d bytes", dataLen)
		}
		pkt.Data = make([]byte, dataLen)
		if _, err := io.ReadFull(r, pkt.Data); err != nil {
			return Packet{}, err
		}
	}

	return pkt, nil
}
