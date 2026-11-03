package nmt

import (
	"crypto/sha256"
)

const (
	LeafPrefix byte = 0x00
	NodePrefix byte = 0x01
)

// Node represents an NMT tree node containing namespace bounds and a SHA-256 digest.
type Node struct {
	MinNS  NamespaceID
	MaxNS  NamespaceID
	Digest [32]byte
}

// HashLeaf computes the leaf node hash: SHA256(0x00 || nid || nid || data).
func HashLeaf(nid NamespaceID, data []byte) Node {
	h := sha256.New()
	h.Write([]byte{LeafPrefix})
	h.Write(nid[:])
	h.Write(nid[:])
	h.Write(data)

	var digest [32]byte
	copy(digest[:], h.Sum(nil))

	return Node{
		MinNS:  nid,
		MaxNS:  nid,
		Digest: digest,
	}
}

// HashNode combines two child nodes into a parent node:
// MinNS = min(left.MinNS, right.MinNS)
// MaxNS = max(left.MaxNS, right.MaxNS)
// Digest = SHA256(0x01 || MinNS || MaxNS || left.Bytes() || right.Bytes())
func HashNode(left, right Node) Node {
	minNS := left.MinNS
	if right.MinNS.Less(minNS) {
		minNS = right.MinNS
	}

	maxNS := left.MaxNS
	if maxNS.Less(right.MaxNS) {
		maxNS = right.MaxNS
	}

	h := sha256.New()
	h.Write([]byte{NodePrefix})
	h.Write(minNS[:])
	h.Write(maxNS[:])
	h.Write(left.Bytes())
	h.Write(right.Bytes())

	var digest [32]byte
	copy(digest[:], h.Sum(nil))

	return Node{
		MinNS:  minNS,
		MaxNS:  maxNS,
		Digest: digest,
	}
}

// Bytes returns the serialized 48-byte representation: MinNS (8) || MaxNS (8) || Digest (32).
func (n Node) Bytes() []byte {
	buf := make([]byte, 48)
	copy(buf[0:8], n.MinNS[:])
	copy(buf[8:16], n.MaxNS[:])
	copy(buf[16:48], n.Digest[:])
	return buf
}

// Equals checks structural and cryptographic equality of two nodes.
func (n Node) Equals(other Node) bool {
	return n.MinNS == other.MinNS && n.MaxNS == other.MaxNS && n.Digest == other.Digest
}
