package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewID returns a random RFC 4122 version 4 UUID. It panics only if the
// operating system cannot supply randomness, which is unrecoverable.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("uuid: read random bytes: %w", err))
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // variant 10
	var s [36]byte
	hex.Encode(s[0:8], b[0:4])
	s[8] = '-'
	hex.Encode(s[9:13], b[4:6])
	s[13] = '-'
	hex.Encode(s[14:18], b[6:8])
	s[18] = '-'
	hex.Encode(s[19:23], b[8:10])
	s[23] = '-'
	hex.Encode(s[24:], b[10:])
	return string(s[:])
}

// UUIDGenerator generates version 4 UUIDs; it implements app.IDGenerator.
type UUIDGenerator struct{}

// NewID implements app.IDGenerator.
func (UUIDGenerator) NewID() string { return NewID() }
