package nmt

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// NamespaceSize defines the fixed byte length of namespace identifiers (8 bytes).
const NamespaceSize = 8

// NamespaceID is an 8-byte identifier defining data partitioning in the NMT.
type NamespaceID [NamespaceSize]byte

// MinNamespace returns the minimum possible namespace (all 0x00).
func MinNamespace() NamespaceID {
	return NamespaceID{}
}

// MaxNamespace returns the maximum possible namespace (all 0xFF).
func MaxNamespace() NamespaceID {
	var ns NamespaceID
	for i := range ns {
		ns[i] = 0xFF
	}
	return ns
}

// NewNamespaceFromUint64 constructs a NamespaceID from a uint64 integer.
func NewNamespaceFromUint64(id uint64) NamespaceID {
	var ns NamespaceID
	binary.BigEndian.PutUint64(ns[:], id)
	return ns
}

// ToUint64 converts the NamespaceID to a uint64 integer.
func (ns NamespaceID) ToUint64() uint64 {
	return binary.BigEndian.Uint64(ns[:])
}

// Compare returns -1 if ns < other, 0 if ns == other, 1 if ns > other.
func (ns NamespaceID) Compare(other NamespaceID) int {
	return bytes.Compare(ns[:], other[:])
}

// Less reports whether ns < other.
func (ns NamespaceID) Less(other NamespaceID) bool {
	return ns.Compare(other) < 0
}

// Equals reports whether ns == other.
func (ns NamespaceID) Equals(other NamespaceID) bool {
	return ns == other
}

// ValidateOrder verifies that slice of namespaces is monotonically sorted.
func ValidateOrder(namespaces []NamespaceID) error {
	for i := 1; i < len(namespaces); i++ {
		if namespaces[i].Less(namespaces[i-1]) {
			return errors.New("nmt: namespaces are not in non-decreasing order")
		}
	}
	return nil
}
