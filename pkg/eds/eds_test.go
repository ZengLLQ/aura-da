package eds_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
	"github.com/ZengLLQ/aura-da/pkg/reedsolomon"
)

func TestExtendedDataSquareGeneration(t *testing.T) {
	k := 4
	var originalShares []eds.Share

	// Build 16 sorted original shares
	for r := 0; r < k; r++ {
		for c := 0; c < k; c++ {
			ns := nmt.NewNamespaceFromUint64(uint64(r*k + c + 1))
			payload := []byte(fmt.Sprintf("block_tx_payload_r%d_c%d_padbytes", r, c))
			originalShares = append(originalShares, eds.NewShare(ns, payload))
		}
	}

	sq, err := eds.NewSquare(k, originalShares)
	if err != nil {
		t.Fatalf("failed to create square: %v", err)
	}

	if sq.ExtendedWidth != 8 {
		t.Fatalf("expected extended width 8, got %d", sq.ExtendedWidth)
	}

	// 1. Verify Q1 matches original shares verbatim
	for r := 0; r < k; r++ {
		for c := 0; c < k; c++ {
			orig := originalShares[r*k+c]
			cell := sq.Shares[r][c]
			if !orig.Namespace.Equals(cell.Namespace) || !bytes.Equal(orig.Data, cell.Data) {
				t.Fatalf("cell (%d, %d) does not match original", r, c)
			}
		}
	}

	// 2. Verify all rows and columns satisfy Reed-Solomon parity
	rs, _ := reedsolomon.New(k, k)
	for r := 0; r < sq.ExtendedWidth; r++ {
		rowData := make([][]byte, sq.ExtendedWidth)
		for c := 0; c < sq.ExtendedWidth; c++ {
			rowData[c] = sq.Shares[r][c].Data
		}
		ok, err := rs.Verify(rowData)
		if err != nil || !ok {
			t.Fatalf("row %d failed RS parity verification", r)
		}
	}

	for c := 0; c < sq.ExtendedWidth; c++ {
		colData := make([][]byte, sq.ExtendedWidth)
		for r := 0; r < sq.ExtendedWidth; r++ {
			colData[r] = sq.Shares[r][c].Data
		}
		ok, err := rs.Verify(colData)
		if err != nil || !ok {
			t.Fatalf("column %d failed RS parity verification", c)
		}
	}

	// 3. Verify DAH computation
	dah, err := eds.ComputeDAH(sq)
	if err != nil {
		t.Fatalf("failed ComputeDAH: %v", err)
	}
	if len(dah.RowRoots) != 8 || len(dah.ColRoots) != 8 {
		t.Fatalf("DAH dimension mismatch: %d rows, %d cols", len(dah.RowRoots), len(dah.ColRoots))
	}
	rootHash := dah.RootHash()
	if rootHash == [32]byte{} {
		t.Fatal("DAH root hash is zero")
	}
}
