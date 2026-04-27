// Package codec provides FIX protocol encoding and decoding capabilities.
//
// Build modes:
//   - Source mode (default): go build
//     Compiles C sources directly with cgo
//   - Prebuilt mode: go build -tags lib
//     Links against prebuilt static library
//
// The cgo configuration is in cgo_source.go and cgo_prebuilt.go
package codec

/*
#include "codec.h"
#include "fix_codec.h"
#include <stdlib.h>

// Forward declarations for fc_init/fc_cleanup (no public header)
int fc_init(void);
void fc_cleanup(void);
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// Config holds library configuration options
type Config struct {
	NumThreads    int
	SIMDLevel     string
	EnableAVX2    bool
	MemoryLimitMB uint64
	Verbose       bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		NumThreads:    runtime.NumCPU(),
		SIMDLevel:     "auto",
		EnableAVX2:    true,
		MemoryLimitMB: 0,
		Verbose:       false,
	}
}

type libState struct {
	refCount int
	mu       sync.Mutex
}

var state = &libState{}

// Init initializes the library with default configuration
func Init() error {
	return InitWithConfig(DefaultConfig())
}

// InitWithConfig initializes the library with custom configuration
func InitWithConfig(cfg Config) error {
	state.mu.Lock()
	defer state.mu.Unlock()

	// Allow multiple Init calls (reference counting)
	if state.refCount > 0 {
		state.refCount++
		return nil
	}

	status := C.fc_init()
	if status != 0 {
		return fmt.Errorf("initialization failed with status %d", status)
	}

	state.refCount = 1
	return nil
}

// Cleanup releases library resources
func Cleanup() {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.refCount <= 0 {
		return
	}

	state.refCount--
	if state.refCount == 0 {
		C.fc_cleanup()
	}
}

// IsInitialized returns whether the library is initialized
func IsInitialized() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.refCount > 0
}

// SIMDLevel returns the current SIMD acceleration level
func SIMDLevel() string {
	return C.GoString(C.fc_fix_simd_level())
}

// FIXSection represents field section classification
type FIXSection uint8

const (
	FIXSectionHeader  FIXSection = C.FC_FIX_SECTION_HEADER
	FIXSectionBody    FIXSection = C.FC_FIX_SECTION_BODY
	FIXSectionTrailer FIXSection = C.FC_FIX_SECTION_TRAILER
)

// FIXParsedField represents a parsed FIX field
type FIXParsedField struct {
	Tag         int32
	ValueOffset uint32
	ValueLength uint16
	RawOffset   uint32
	RawLength   uint16
	Section     FIXSection
}

// FIXParsedMessage represents a parsed FIX message
type FIXParsedMessage struct {
	MsgOffset           uint32
	MsgLength           uint32
	ComputedChecksum    uint32
	ComputedBodyLength  uint32
	DeclaredChecksum    uint32
	DeclaredBodyLength  uint32
	FieldCount          int32
	HeaderFieldCount    int32
	BodyFieldCount      int32
	TrailerFieldCount   int32
	RequiredTagsBitmap  uint32
	Fields              []FIXParsedField
}

// FIXParseResult represents batch parse result
type FIXParseResult struct {
	MsgCount      int32
	ConsumedBytes int32
	Messages      []FIXParsedMessage
}

// ParseFIXMessages parses FIX messages from raw byte stream (batch mode)
func ParseFIXMessages(data []byte) (*FIXParseResult, error) {
	if len(data) == 0 {
		return &FIXParseResult{}, nil
	}

	var result C.fc_fix_parse_result_t
	count := C.fc_fix_parse_messages(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&result,
	)

	if count < 0 {
		return nil, fmt.Errorf("parse failed with code %d", count)
	}

	// Convert C result to Go
	goResult := &FIXParseResult{
		MsgCount:      int32(result.msg_count),
		ConsumedBytes: int32(result.consumed_bytes),
		Messages:      make([]FIXParsedMessage, int32(result.msg_count)),
	}

	for i := int32(0); i < int32(result.msg_count); i++ {
		msg := &result.messages[i]
		goMsg := &goResult.Messages[i]

		goMsg.MsgOffset = uint32(msg.msg_offset)
		goMsg.MsgLength = uint32(msg.msg_length)
		goMsg.ComputedChecksum = uint32(msg.computed_checksum)
		goMsg.ComputedBodyLength = uint32(msg.computed_body_length)
		goMsg.DeclaredChecksum = uint32(msg.declared_checksum)
		goMsg.DeclaredBodyLength = uint32(msg.declared_body_length)
		goMsg.FieldCount = int32(msg.field_count)
		goMsg.HeaderFieldCount = int32(msg.header_field_count)
		goMsg.BodyFieldCount = int32(msg.body_field_count)
		goMsg.TrailerFieldCount = int32(msg.trailer_field_count)
		goMsg.RequiredTagsBitmap = uint32(msg.required_tags_bitmap)

		goMsg.Fields = make([]FIXParsedField, int32(msg.field_count))
		for j := int32(0); j < int32(msg.field_count); j++ {
			field := &msg.fields[j]
			goMsg.Fields[j] = FIXParsedField{
				Tag:         int32(field.tag),
				ValueOffset: uint32(field.value_offset),
				ValueLength: uint16(field.value_length),
				RawOffset:   uint32(field.raw_offset),
				RawLength:   uint16(field.raw_length),
				Section:     FIXSection(field.section),
			}
		}
	}

	return goResult, nil
}

// ParseFIXOne parses a single FIX message (lightweight version)
func ParseFIXOne(data []byte) (*FIXParsedMessage, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var result C.fc_fix_parse_single_result_t
	success := C.fc_fix_parse_one(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&result,
	)

	if success == 0 {
		return nil, nil
	}

	// Convert C result to Go
	goMsg := &FIXParsedMessage{
		MsgOffset:          uint32(result.msg_offset),
		MsgLength:          uint32(result.msg_length),
		ComputedChecksum:   uint32(result.computed_checksum),
		ComputedBodyLength: uint32(result.computed_body_length),
		DeclaredChecksum:   uint32(result.declared_checksum),
		DeclaredBodyLength: uint32(result.declared_body_length),
		FieldCount:         int32(result.field_count),
		HeaderFieldCount:   int32(result.header_field_count),
		BodyFieldCount:     int32(result.body_field_count),
		TrailerFieldCount:  int32(result.trailer_field_count),
		RequiredTagsBitmap: uint32(result.required_tags_bitmap),
		Fields:             make([]FIXParsedField, int32(result.field_count)),
	}

	for i := int32(0); i < int32(result.field_count); i++ {
		field := &result.fields[i]
		goMsg.Fields[i] = FIXParsedField{
			Tag:         int32(field.tag),
			ValueOffset: uint32(field.value_offset),
			ValueLength: uint16(field.value_length),
			RawOffset:   uint32(field.raw_offset),
			RawLength:   uint16(field.raw_length),
			Section:     FIXSection(field.section),
		}
	}

	return goMsg, nil
}

// FIXMarshalField represents a field for marshaling
type FIXMarshalField struct {
	Tag   int32
	Value []byte
}

// MarshalFIXMessage assembles structured fields into complete FIX message
func MarshalFIXMessage(beginString string, fields []FIXMarshalField, outputBuf []byte) (int, error) {
	if len(outputBuf) == 0 {
		return 0, fmt.Errorf("output buffer is empty")
	}

	// Allocate C memory for fields to avoid Go pointer issues
	cFields := (*C.fc_fix_marshal_field_t)(C.malloc(C.size_t(len(fields)) * C.size_t(unsafe.Sizeof(C.fc_fix_marshal_field_t{}))))
	if cFields == nil {
		return 0, fmt.Errorf("failed to allocate memory for fields")
	}
	defer C.free(unsafe.Pointer(cFields))

	// Convert to slice for easier access
	cFieldsSlice := unsafe.Slice(cFields, len(fields))

	// Track allocated C memory for field values
	var allocatedPtrs []unsafe.Pointer

	for i, field := range fields {
		cFieldsSlice[i].tag = C.int32_t(field.Tag)
		if len(field.Value) > 0 {
			// Allocate C memory for value
			cValue := C.CBytes(field.Value)
			allocatedPtrs = append(allocatedPtrs, cValue)
			cFieldsSlice[i].value = (*C.uint8_t)(cValue)
			cFieldsSlice[i].value_length = C.uint16_t(len(field.Value))
		}
	}

	// Free allocated C memory for field values
	defer func() {
		for _, ptr := range allocatedPtrs {
			C.free(ptr)
		}
	}()

	beginStringBytes := []byte(beginString)

	result := C.fc_fix_marshal_message(
		(*C.uint8_t)(unsafe.Pointer(&beginStringBytes[0])),
		C.int32_t(len(beginStringBytes)),
		cFields,
		C.int32_t(len(fields)),
		(*C.uint8_t)(unsafe.Pointer(&outputBuf[0])),
		C.size_t(len(outputBuf)),
	)

	if result.success == 0 {
		return 0, fmt.Errorf("marshal failed")
	}

	return int(result.output_length), nil
}

// FIXChecksum calculates byte sum for FIX CheckSum
func FIXChecksum(data []byte) uint32 {
	if len(data) == 0 {
		return 0
	}

	return uint32(C.fc_fix_checksum(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	))
}
