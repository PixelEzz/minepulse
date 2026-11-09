package slp_test

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"

	"github.com/PixelEzz/minepulse/internal/slp"
)

func TestVarIntEncodingDecoding(t *testing.T) {
	testCases := []struct {
		name  string
		value int32
		bytes []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"two", 2, []byte{0x02}},
		{"127", 127, []byte{0x7f}},
		{"128", 128, []byte{0x80, 0x01}},
		{"255", 255, []byte{0xff, 0x01}},
		{"2147483647 (max int32)", math.MaxInt32, []byte{0xff, 0xff, 0xff, 0xff, 0x07}},
		{"-1", -1, []byte{0xff, 0xff, 0xff, 0xff, 0x0f}},
		{"-2147483648 (min int32)", math.MinInt32, []byte{0x80, 0x80, 0x80, 0x80, 0x08}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := slp.WriteVarInt(buf, tc.value)
			if err != nil {
				t.Fatalf("WriteVarInt failed for %d: %v", tc.value, err)
			}

			if !bytes.Equal(buf.Bytes(), tc.bytes) {
				t.Errorf("encoded bytes mismatch for %d: got %x, expected %x", tc.value, buf.Bytes(), tc.bytes)
			}

			decoded, err := slp.ReadVarInt(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("ReadVarInt failed for %d: %v", tc.value, err)
			}

			if decoded != tc.value {
				t.Errorf("decoded value mismatch: got %d, expected %d", decoded, tc.value)
			}
		})
	}
}

func TestVarIntBufferOverflow(t *testing.T) {
	// A 6-byte sequence with MSB set in all bytes should return ErrVarIntTooBig
	malformed := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	_, err := slp.ReadVarInt(bytes.NewReader(malformed))
	if err == nil {
		t.Fatal("expected error for 6-byte varint, got nil")
	}
}

func TestStringEncodeDecode(t *testing.T) {
	testStr := "§6MinePulse §aMinecraft SLP Test String 123!"
	buf := new(bytes.Buffer)

	if err := slp.WriteString(buf, testStr); err != nil {
		t.Fatalf("WriteString error: %v", err)
	}

	readBack, err := slp.ReadString(buf, 0)
	if err != nil {
		t.Fatalf("ReadString error: %v", err)
	}

	if readBack != testStr {
		t.Errorf("string mismatch: got %q, expected %q", readBack, testStr)
	}
}

func TestParseMOTD(t *testing.T) {
	// Case 1: Plain string with formatting
	rawPlain := json.RawMessage(`"§aWelcome to §bMinePulse Server"`)
	motd := slp.ParseMOTD(rawPlain)
	if motd != "Welcome to MinePulse Server" {
		t.Errorf("unexpected clean MOTD: got %q", motd)
	}

	// Case 2: Chat component JSON
	rawComponent := json.RawMessage(`{
		"text": "§eHypixel §6Network ",
		"extra": [
			{"text": "§c[1.8-1.20]"}
		]
	}`)
	motdComp := slp.ParseMOTD(rawComponent)
	if motdComp != "Hypixel Network [1.8-1.20]" {
		t.Errorf("unexpected component MOTD: got %q", motdComp)
	}
}
