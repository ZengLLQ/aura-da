package eds

import (
	"errors"

	"github.com/ZengLLQ/aura-da/pkg/nmt"
)

// Share represents an atomic data chunk in the Extended Data Square.
// Each share contains an 8-byte namespace prefix followed by raw payload bytes.
type Share struct {
	Namespace nmt.NamespaceID
	Data      []byte
}

// NewShare creates a new Share instance.
func NewShare(ns nmt.NamespaceID, data []byte) Share {
	d := make([]byte, len(data))
	copy(d, data)
	return Share{
		Namespace: ns,
		Data:      d,
	}
}

// ToBytes returns the serialized share: NamespaceID (8) || Data.
func (s Share) ToBytes() []byte {
	buf := make([]byte, nmt.NamespaceSize+len(s.Data))
	copy(buf[:nmt.NamespaceSize], s.Namespace[:])
	copy(buf[nmt.NamespaceSize:], s.Data)
	return buf
}

// FromBytes parses a raw byte slice into a Share.
func FromBytes(buf []byte) (Share, error) {
	if len(buf) < nmt.NamespaceSize {
		return Share{}, errors.New("eds: buffer too small to contain namespace")
	}
	var ns nmt.NamespaceID
	copy(ns[:], buf[:nmt.NamespaceSize])
	data := make([]byte, len(buf)-nmt.NamespaceSize)
	copy(data, buf[nmt.NamespaceSize:])
	return Share{
		Namespace: ns,
		Data:      data,
	}, nil
}
