package slp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	ErrVarIntTooBig = errors.New("varint is too big (exceeds 5 bytes)")
	ErrStringTooLong = errors.New("string length exceeds protocol limit")
)

// WriteVarInt encodes an int32 as a Minecraft VarInt into w.
func WriteVarInt(w io.Writer, value int32) error {
	uval := uint32(value)
	var buf [5]byte
	idx := 0
	for {
		b := byte(uval & 0x7F)
		uval >>= 7
		if uval != 0 {
			b |= 0x80
		}
		buf[idx] = b
		idx++
		if uval == 0 {
			break
		}
	}
	_, err := w.Write(buf[:idx])
	if err != nil {
		return fmt.Errorf("failed to write varint: %w", err)
	}
	return nil
}

// ReadVarInt decodes a Minecraft VarInt from r.
func ReadVarInt(r io.Reader) (int32, error) {
	var numRead int
	var result uint32
	var b [1]byte

	for {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, fmt.Errorf("failed to read varint byte: %w", err)
		}
		read := b[0]
		value := uint32(read & 0x7F)
		result |= value << (7 * numRead)
		numRead++
		if numRead > 5 {
			return 0, ErrVarIntTooBig
		}
		if (read & 0x80) == 0 {
			break
		}
	}

	return int32(result), nil
}

// WriteString writes a UTF-8 string prefixed by its VarInt byte length.
func WriteString(w io.Writer, s string) error {
	b := []byte(s)
	if err := WriteVarInt(w, int32(len(b))); err != nil {
		return fmt.Errorf("failed writing string length: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("failed writing string body: %w", err)
	}
	return nil
}

// ReadString reads a UTF-8 string prefixed by a VarInt byte length.
// maxLen limits the allowed byte count (0 means default 32767).
func ReadString(r io.Reader, maxLen int) (string, error) {
	if maxLen <= 0 {
		maxLen = 32767
	}
	length, err := ReadVarInt(r)
	if err != nil {
		return "", fmt.Errorf("failed reading string length: %w", err)
	}
	if length < 0 || int(length) > maxLen {
		return "", fmt.Errorf("%w: length %d exceeds max %d", ErrStringTooLong, length, maxLen)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("failed reading string payload: %w", err)
	}
	return string(buf), nil
}

// WritePacket encapsulates packetID and payload in a VarInt length-prefixed frame.
func WritePacket(w io.Writer, packetID int32, payload []byte) error {
	var body bytes.Buffer
	if err := WriteVarInt(&body, packetID); err != nil {
		return fmt.Errorf("failed to encode packet ID: %w", err)
	}
	if len(payload) > 0 {
		if _, err := body.Write(payload); err != nil {
			return fmt.Errorf("failed to write payload: %w", err)
		}
	}

	packetBytes := body.Bytes()
	if err := WriteVarInt(w, int32(len(packetBytes))); err != nil {
		return fmt.Errorf("failed to write packet length: %w", err)
	}
	if _, err := w.Write(packetBytes); err != nil {
		return fmt.Errorf("failed to send packet bytes: %w", err)
	}
	return nil
}

// ReadPacket reads a length-prefixed packet frame, returns packet ID and payload bytes.
func ReadPacket(r io.Reader) (int32, []byte, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read packet length: %w", err)
	}
	if length <= 0 || length > 2097151 { // 2MB safety bound for standard Minecraft packets
		return 0, nil, fmt.Errorf("invalid packet length: %d", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return 0, nil, fmt.Errorf("failed to read packet data: %w", err)
	}

	buf := bytes.NewReader(data)
	packetID, err := ReadVarInt(buf)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read packet id from payload: %w", err)
	}

	payload := data[len(data)-buf.Len():]
	return packetID, payload, nil
}
