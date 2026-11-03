package nmt

import (
	"errors"
	"fmt"
)

// Proof contains cryptographic audit co-path nodes proving inclusion or absence.
type Proof struct {
	StartIdx   int    // Starting leaf index in tree
	EndIdx     int    // Ending leaf index in tree (exclusive)
	Nodes      []Node // Audit path sibling nodes
	IsAbsence  bool   // Whether this proves non-existence of the namespace
	AbsencePos int    // For absence proof: index of adjacent leaf
}

// ProveNamespace generates an inclusion proof for all leaves with namespace nid,
// or an absence proof if nid does not exist in the tree.
func (t *Tree) ProveNamespace(nid NamespaceID) (Proof, error) {
	if len(t.leaves) == 0 {
		return Proof{}, errors.New("nmt: cannot generate proof on empty tree")
	}

	start := -1
	end := -1
	for i, ns := range t.leafNamespaces {
		if ns.Equals(nid) {
			if start == -1 {
				start = i
			}
			end = i + 1
		}
	}

	// If namespace is not present, generate an absence proof
	if start == -1 {
		// Find insertion index where nid would belong
		pos := 0
		for pos < len(t.leafNamespaces) && t.leafNamespaces[pos].Less(nid) {
			pos++
		}
		// Prove leaf at pos (or pos-1)
		targetLeaf := pos
		if targetLeaf >= len(t.leaves) {
			targetLeaf = len(t.leaves) - 1
		}

		coPath := t.buildCoPathForRange(targetLeaf, targetLeaf+1)
		return Proof{
			StartIdx:   targetLeaf,
			EndIdx:     targetLeaf + 1,
			Nodes:      coPath,
			IsAbsence:  true,
			AbsencePos: targetLeaf,
		}, nil
	}

	coPath := t.buildCoPathForRange(start, end)
	return Proof{
		StartIdx:   start,
		EndIdx:     end,
		Nodes:      coPath,
		IsAbsence:  false,
		AbsencePos: 0,
	}, nil
}

// ProveIndex generates an inclusion proof for a single leaf index.
func (t *Tree) ProveIndex(idx int) (Proof, error) {
	if idx < 0 || idx >= len(t.leaves) {
		return Proof{}, fmt.Errorf("nmt: leaf index %d out of bounds [0, %d)", idx, len(t.leaves))
	}
	coPath := t.buildCoPathForRange(idx, idx+1)
	return Proof{
		StartIdx:  idx,
		EndIdx:    idx + 1,
		Nodes:     coPath,
		IsAbsence: false,
	}, nil
}

// buildCoPathForRange traverses binary tree levels to collect audit co-path sibling hashes.
func (t *Tree) buildCoPathForRange(start, end int) []Node {
	var coPath []Node
	currentLevel := make([]Node, len(t.leaves))
	copy(currentLevel, t.leaves)

	curStart := start
	curEnd := end

	for len(currentLevel) > 1 {
		// Left sibling if curStart is odd
		if curStart%2 == 1 {
			coPath = append(coPath, currentLevel[curStart-1])
		}
		// Right sibling if curEnd is odd and curEnd < len(currentLevel)
		if curEnd%2 == 1 && curEnd < len(currentLevel) {
			coPath = append(coPath, currentLevel[curEnd])
		}

		// Advance to parent level
		var nextLevel []Node
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				nextLevel = append(nextLevel, HashNode(currentLevel[i], currentLevel[i+1]))
			} else {
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}

		curStart /= 2
		curEnd = (curEnd + 1) / 2
		currentLevel = nextLevel
	}

	return coPath
}

// VerifyInclusion verifies that leaves belonging to nid match root via proof.
func VerifyInclusion(root Node, nid NamespaceID, leavesData [][]byte, proof Proof) bool {
	if proof.IsAbsence || len(leavesData) != (proof.EndIdx-proof.StartIdx) {
		return false
	}

	// Reconstruct leaf nodes
	currentNodes := make([]Node, len(leavesData))
	for i, d := range leavesData {
		currentNodes[i] = HashLeaf(nid, d)
	}

	return verifyCoPath(root, currentNodes, proof.StartIdx, proof.EndIdx, proof.Nodes)
}

// VerifyLeafIndex verifies a single leaf's inclusion by its exact index.
func VerifyLeafIndex(root Node, nid NamespaceID, data []byte, proof Proof) bool {
	if proof.IsAbsence || (proof.EndIdx-proof.StartIdx) != 1 {
		return false
	}
	leafNode := HashLeaf(nid, data)
	return verifyCoPath(root, []Node{leafNode}, proof.StartIdx, proof.EndIdx, proof.Nodes)
}

// VerifyAbsence verifies that namespace nid is absent from the tree.
func VerifyAbsence(root Node, nid NamespaceID, adjacentData []byte, adjacentNS NamespaceID, proof Proof) bool {
	if !proof.IsAbsence {
		return false
	}
	// The adjacent leaf cannot equal the queried namespace
	if adjacentNS.Equals(nid) {
		return false
	}
	leafNode := HashLeaf(adjacentNS, adjacentData)
	return verifyCoPath(root, []Node{leafNode}, proof.StartIdx, proof.EndIdx, proof.Nodes)
}

func verifyCoPath(root Node, currentNodes []Node, start, end int, coPath []Node) bool {
	pathIdx := 0
	curStart := start
	curEnd := end

	for len(currentNodes) > 1 || pathIdx < len(coPath) {
		// If curStart is odd, prepend left sibling from coPath
		if curStart%2 == 1 {
			if pathIdx >= len(coPath) {
				return false
			}
			currentNodes = append([]Node{coPath[pathIdx]}, currentNodes...)
			pathIdx++
			curStart--
		}
		// If curEnd is odd, append right sibling from coPath
		if curEnd%2 == 1 && pathIdx < len(coPath) {
			currentNodes = append(currentNodes, coPath[pathIdx])
			pathIdx++
			curEnd++
		}

		// Compute parent level
		var nextLevel []Node
		for i := 0; i < len(currentNodes); i += 2 {
			if i+1 < len(currentNodes) {
				nextLevel = append(nextLevel, HashNode(currentNodes[i], currentNodes[i+1]))
			} else {
				nextLevel = append(nextLevel, currentNodes[i])
			}
		}

		currentNodes = nextLevel
		curStart /= 2
		curEnd = (curEnd + 1) / 2

		if len(currentNodes) == 1 && pathIdx == len(coPath) {
			break
		}
	}

	if len(currentNodes) != 1 {
		return false
	}
	return currentNodes[0].Equals(root)
}
