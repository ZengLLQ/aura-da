package eds

import (
	"crypto/sha256"
	"fmt"

	"github.com/ZengLLQ/aura-da/pkg/nmt"
)

// DataAvailabilityHeader (DAH) represents the commitment to the 2D Extended Data Square.
// It contains the NMT roots of all rows and columns.
type DataAvailabilityHeader struct {
	RowRoots []nmt.Node
	ColRoots []nmt.Node
}

// ComputeDAH generates the DataAvailabilityHeader from an Extended Data Square.
func ComputeDAH(sq *Square) (*DataAvailabilityHeader, error) {
	w := sq.ExtendedWidth
	rowRoots := make([]nmt.Node, w)
	colRoots := make([]nmt.Node, w)

	// Compute row NMT roots
	for r := 0; r < w; r++ {
		tree := nmt.New()
		rowShares := sq.Row(r)
		for _, sh := range rowShares {
			if err := tree.Push(sh.Namespace, sh.Data); err != nil {
				return nil, fmt.Errorf("dah: row %d push error: %w", r, err)
			}
		}
		root, err := tree.Root()
		if err != nil {
			return nil, fmt.Errorf("dah: row %d root error: %w", r, err)
		}
		rowRoots[r] = root
	}

	// Compute column NMT roots
	for c := 0; c < w; c++ {
		tree := nmt.New()
		colShares := sq.Col(c)
		for _, sh := range colShares {
			if err := tree.Push(sh.Namespace, sh.Data); err != nil {
				return nil, fmt.Errorf("dah: col %d push error: %w", c, err)
			}
		}
		root, err := tree.Root()
		if err != nil {
			return nil, fmt.Errorf("dah: col %d root error: %w", c, err)
		}
		colRoots[c] = root
	}

	return &DataAvailabilityHeader{
		RowRoots: rowRoots,
		ColRoots: colRoots,
	}, nil
}

// RootHash computes the top-level 32-byte cryptographic root hash of the DAH.
func (dah *DataAvailabilityHeader) RootHash() [32]byte {
	h := sha256.New()
	for _, r := range dah.RowRoots {
		h.Write(r.Bytes())
	}
	for _, c := range dah.ColRoots {
		h.Write(c.Bytes())
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}
