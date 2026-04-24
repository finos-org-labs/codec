package codec

import (
	"testing"
)

func TestInit(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Cleanup()

	if !IsInitialized() {
		t.Error("Library should be initialized")
	}
}

func TestSIMDLevel(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Cleanup()

	level := SIMDLevel()
	if level == "" {
		t.Error("SIMD level should not be empty")
	}
	t.Logf("SIMD level: %s", level)
}

func TestFIXChecksum(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Cleanup()

	data := []byte("8=FIX.4.4\x019=100\x01")
	checksum := FIXChecksum(data)
	if checksum == 0 {
		t.Error("Checksum should not be zero for non-empty data")
	}
	t.Logf("Checksum: %d", checksum)
}

func TestParseFIXOne(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Cleanup()

	// Simple FIX message
	fixMsg := "8=FIX.4.4\x019=100\x0135=D\x0149=SENDER\x0156=TARGET\x0134=1\x0152=20240101-00:00:00\x0110=000\x01"

	result, err := ParseFIXOne([]byte(fixMsg))
	if err != nil {
		t.Fatalf("ParseFIXOne failed: %v", err)
	}

	if result == nil {
		t.Log("No complete message found (expected for test message)")
		return
	}

	t.Logf("Parsed message: %d fields", result.FieldCount)
}

func TestParseFIXMessages(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Cleanup()

	// Simple FIX message
	fixMsg := "8=FIX.4.4\x019=100\x0135=D\x0149=SENDER\x0156=TARGET\x0134=1\x0152=20240101-00:00:00\x0110=000\x01"

	result, err := ParseFIXMessages([]byte(fixMsg))
	if err != nil {
		t.Fatalf("ParseFIXMessages failed: %v", err)
	}

	t.Logf("Parsed %d messages, consumed %d bytes", result.MsgCount, result.ConsumedBytes)
}

func TestMarshalFIXMessage(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Cleanup()

	fields := []FIXMarshalField{
		{Tag: 35, Value: []byte("D")},
		{Tag: 49, Value: []byte("SENDER")},
		{Tag: 56, Value: []byte("TARGET")},
		{Tag: 34, Value: []byte("1")},
		{Tag: 52, Value: []byte("20240101-00:00:00")},
	}

	outputBuf := make([]byte, 1024)
	length, err := MarshalFIXMessage("FIX.4.4", fields, outputBuf)
	if err != nil {
		t.Fatalf("MarshalFIXMessage failed: %v", err)
	}

	if length == 0 {
		t.Error("Marshal should produce non-zero output")
	}

	t.Logf("Marshaled message length: %d", length)
	t.Logf("Marshaled message: %q", outputBuf[:length])
}
