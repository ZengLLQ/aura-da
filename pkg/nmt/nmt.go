package nmt

import (
	"errors"
	"fmt"
)

// Tree represents a Namespaced Merkle Tree.
type Tree struct {
	leaves         []Node
	leafData       [][]byte
	leafNamespaces []NamespaceID
	cachedRoot     *Node
}

// New creates an empty Namespaced Merkle Tree.
func New() *Tree {
	return &Tree{}
}

// Push adds a new leaf to the tree. Leaves MUST be pushed in non-decreasing order of namespace.
func (t *Tree) Push(nid NamespaceID, data []byte) error {
	if len(t.leaves) > 0 {
		lastNS := t.leafNamespaces[len(t.leafNamespaces)-1]
		if nid.Less(lastNS) {
			return fmt.Errorf("nmt: out-of-order namespace: cannot push %x after %x", nid[:], lastNS[:])
		}
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	leafNode := HashLeaf(nid, dataCopy)
	t.leaves = append(t.leaves, leafNode)
	t.leafData = append(t.leafData, dataCopy)
	t.leafNamespaces = append(t.leafNamespaces, nid)
	t.cachedRoot = nil

	return nil
}

// LeafCount returns the number of leaves in the tree.
func (t *Tree) LeafCount() int {
	return len(t.leaves)
}

// Leaves returns a copy of the leaf nodes.
func (t *Tree) Leaves() []Node {
	out := make([]Node, len(t.leaves))
	copy(out, t.leaves)
	return out
}

// Root computes and returns the cryptographic NMT root node.
func (t *Tree) Root() (Node, error) {
	if len(t.leaves) == 0 {
		return Node{}, errors.New("nmt: cannot compute root of empty tree")
	}
	if t.cachedRoot != nil {
		return *t.cachedRoot, nil
	}

	currentLevel := make([]Node, len(t.leaves))
	copy(currentLevel, t.leaves)

	for len(currentLevel) > 1 {
		var nextLevel []Node
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				parent := HashNode(currentLevel[i], currentLevel[i+1])
				nextLevel = append(nextLevel, parent)
			} else {
				// Odd element promoted to next level
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}
		currentLevel = nextLevel
	}

	root := currentLevel[0]
	t.cachedRoot = &root
	return root, nil
}
