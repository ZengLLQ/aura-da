package fraud_test

import (
	"fmt"
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/fraud"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
)

func TestBadEncodingFraudProof(t *testing.T) {
	k := 4
	var originalShares []eds.Share
	for r := 0; r < k; r++ {
		for c := 0; c < k; c++ {
			ns := nmt.NewNamespaceFromUint64(uint64(r*k + c + 1))
			data := []byte(fmt.Sprintf("block_tx_data_payload_r%02d_c%02d", r, c))
			originalShares = append(originalShares, eds.NewShare(ns, data))
		}
	}

	sq, err := eds.NewSquare(k, originalShares)
	if err != nil {
		t.Fatalf("failed NewSquare: %v", err)
	}

	// Corrupt a parity cell in Row 2, Column 6
	corruptedRow := 2
	corruptedCol := 6
	sq.Shares[corruptedRow][corruptedCol].Data[0] ^= 0xAA

	// Compute DAH committing to the corrupted square
	dah, err := eds.ComputeDAH(sq)
	if err != nil {
		t.Fatalf("ComputeDAH error: %v", err)
	}

	// 1. Generate fraud proof for the corrupted row
	proof, err := fraud.GenerateBadEncodingProof(sq, dah, true, corruptedRow, corruptedCol)
	if err != nil {
		t.Fatalf("GenerateBadEncodingProof error: %v", err)
	}

	// 2. Verify fraud proof (must succeed and confirm invalid encoding)
	isFraud, err := fraud.VerifyBadEncodingProof(proof, dah)
	if err != nil {
		t.Fatalf("VerifyBadEncodingProof error: %v", err)
	}
	if !isFraud {
		t.Fatal("expected fraud proof verification to be true for corrupted row")
	}

	// 3. Test on an uncorrupted row (Row 0)
	cleanProof, err := fraud.GenerateBadEncodingProof(sq, dah, true, 0, 5)
	if err != nil {
		t.Fatalf("clean GenerateBadEncodingProof error: %v", err)
	}
	isCleanFraud, err := fraud.VerifyBadEncodingProof(cleanProof, dah)
	if err != nil {
		t.Fatalf("clean VerifyBadEncodingProof error: %v", err)
	}
	if isCleanFraud {
		t.Fatal("expected fraud proof verification to be false for valid row")
	}
}
