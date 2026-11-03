package fraud

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
	"github.com/ZengLLQ/aura-da/pkg/reedsolomon"
)

// BadEncodingProof proves that a row or column in the 2D EDS was incorrectly encoded.
type BadEncodingProof struct {
	IsRow        bool
	AxisIndex    int
	OriginalK    int
	ShareIndices []int
	Shares       []eds.Share
	ShareProofs  []nmt.Proof
	InvalidIndex int
	InvalidShare eds.Share
	InvalidProof nmt.Proof
}

// GenerateBadEncodingProof creates a compact proof of invalid encoding for row or column axisIndex.
func GenerateBadEncodingProof(
	sq *eds.Square,
	dah *eds.DataAvailabilityHeader,
	isRow bool,
	axisIndex int,
	invalidShareIndex int,
) (*BadEncodingProof, error) {
	k := sq.OriginalWidth
	w := sq.ExtendedWidth

	var axisShares []eds.Share
	if isRow {
		axisShares = sq.Row(axisIndex)
	} else {
		axisShares = sq.Col(axisIndex)
	}

	// Build the tree for the target axis to obtain proofs
	tree := nmt.New()
	for _, sh := range axisShares {
		_ = tree.Push(sh.Namespace, sh.Data)
	}

	// Collect first k shares that do NOT include invalidShareIndex
	var shareIndices []int
	var selectedShares []eds.Share
	var shareProofs []nmt.Proof

	for i := 0; i < w && len(selectedShares) < k; i++ {
		if i == invalidShareIndex {
			continue
		}
		p, err := tree.ProveIndex(i)
		if err != nil {
			return nil, err
		}
		shareIndices = append(shareIndices, i)
		selectedShares = append(selectedShares, axisShares[i])
		shareProofs = append(shareProofs, p)
	}

	if len(selectedShares) < k {
		return nil, errors.New("fraud: insufficient shares available to form bad encoding proof")
	}

	invalidProof, err := tree.ProveIndex(invalidShareIndex)
	if err != nil {
		return nil, err
	}

	return &BadEncodingProof{
		IsRow:        isRow,
		AxisIndex:    axisIndex,
		OriginalK:    k,
		ShareIndices: shareIndices,
		Shares:       selectedShares,
		ShareProofs:  shareProofs,
		InvalidIndex: invalidShareIndex,
		InvalidShare: axisShares[invalidShareIndex],
		InvalidProof: invalidProof,
	}, nil
}

// VerifyBadEncodingProof mathematically verifies the bad encoding fraud proof against the DAH.
func VerifyBadEncodingProof(proof *BadEncodingProof, dah *eds.DataAvailabilityHeader) (bool, error) {
	k := proof.OriginalK
	w := 2 * k

	var root nmt.Node
	if proof.IsRow {
		if proof.AxisIndex < 0 || proof.AxisIndex >= len(dah.RowRoots) {
			return false, errors.New("fraud: row axis index out of bounds")
		}
		root = dah.RowRoots[proof.AxisIndex]
	} else {
		if proof.AxisIndex < 0 || proof.AxisIndex >= len(dah.ColRoots) {
			return false, errors.New("fraud: column axis index out of bounds")
		}
		root = dah.ColRoots[proof.AxisIndex]
	}

	// 1. Verify NMT inclusion for each of the k selected shares
	if len(proof.Shares) != k || len(proof.ShareIndices) != k || len(proof.ShareProofs) != k {
		return false, errors.New("fraud: proof must contain exactly k valid shares and proofs")
	}

	for i := 0; i < k; i++ {
		sh := proof.Shares[i]
		idx := proof.ShareIndices[i]
		p := proof.ShareProofs[i]
		if !nmt.VerifyLeafIndex(root, sh.Namespace, sh.Data, p) {
			return false, fmt.Errorf("fraud: invalid inclusion proof for share at index %d", idx)
		}
	}

	// 2. Verify NMT inclusion of the alleged invalid share
	if !nmt.VerifyLeafIndex(root, proof.InvalidShare.Namespace, proof.InvalidShare.Data, proof.InvalidProof) {
		return false, errors.New("fraud: invalid inclusion proof for alleged bad share")
	}

	// 3. Reconstruct the true original row/column using the k valid shares
	rs, err := reedsolomon.New(k, k)
	if err != nil {
		return false, err
	}

	shards := make([][]byte, w)
	for i := 0; i < k; i++ {
		shards[proof.ShareIndices[i]] = proof.Shares[i].Data
	}

	if err := rs.Reconstruct(shards); err != nil {
		return false, fmt.Errorf("fraud: failed to reconstruct axis: %w", err)
	}

	// 4. Compare expected share at InvalidIndex against committed InvalidShare
	expectedData := shards[proof.InvalidIndex]
	if bytes.Equal(expectedData, proof.InvalidShare.Data) {
		// If expected matches committed, the encoding was actually valid! (Proof rejected)
		return false, nil
	}

	// Proof is valid: the committed share differs from the mathematically correct Reed-Solomon codeword
	return true, nil
}
