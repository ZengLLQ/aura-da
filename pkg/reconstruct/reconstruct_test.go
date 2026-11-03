package reconstruct_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
	"github.com/ZengLLQ/aura-da/pkg/reconstruct"
)

func TestCascadingSquareReconstruction(t *testing.T) {
	k := 4 // 4x4 original -> 8x8 extended (64 cells)
	var originalShares []eds.Share
	for r := 0; r < k; r++ {
		for c := 0; c < k; c++ {
			ns := nmt.NewNamespaceFromUint64(uint64(r*k + c + 1))
			data := []byte(fmt.Sprintf("block_tx_data_payload_r%02d_c%02d", r, c))
			originalShares = append(originalShares, eds.NewShare(ns, data))
		}
	}

	fullSq, err := eds.NewSquare(k, originalShares)
	if err != nil {
		t.Fatalf("failed NewSquare: %v", err)
	}

	pas := reconstruct.NewPartiallyAvailableSquare(k, len(originalShares[0].Data))

	// Adversarial withholding pattern:
	// Fill only the first k=4 rows completely (32 shares out of 64).
	// Rows 4..7 (bottom half) are completely missing (32 shares withheld).
	for r := 0; r < k; r++ {
		for c := 0; c < 2*k; c++ {
			pas.SetShare(r, c, fullSq.Shares[r][c])
		}
	}

	if pas.AvailableCount() != 32 {
		t.Fatalf("expected 32 initial shares, got %d", pas.AvailableCount())
	}

	// In the first pass:
	// Rows 0..3 have 8 shares each (full).
	// Rows 4..7 have 0 shares.
	// But in columns 0..7, each column has exactly k=4 shares from rows 0..3!
	// Therefore, column passes will decode rows 4..7 in one cascade!
	recoveredSq, err := pas.ReconstructCascading()
	if err != nil {
		t.Fatalf("ReconstructCascading failed: %v", err)
	}

	// Verify byte-level equality on all 64 cells
	for r := 0; r < 2*k; r++ {
		for c := 0; c < 2*k; c++ {
			expected := fullSq.Shares[r][c].Data
			actual := recoveredSq.Shares[r][c].Data
			if !bytes.Equal(expected, actual) {
				t.Fatalf("recovered share mismatch at (%d, %d)", r, c)
			}
		}
	}
}
