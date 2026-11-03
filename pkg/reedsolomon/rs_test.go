package reedsolomon_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/reedsolomon"
)

func TestReedSolomonEncodeAndVerify(t *testing.T) {
	k, m := 4, 4
	rs, err := reedsolomon.New(k, m)
	if err != nil {
		t.Fatalf("failed to create ReedSolomon: %v", err)
	}

	shardSize := 64
	shards := make([][]byte, k+m)
	for i := 0; i < k; i++ {
		shards[i] = make([]byte, shardSize)
		for b := 0; b < shardSize; b++ {
			shards[i][b] = byte((i*17 + b*31) % 256)
		}
	}
	for i := k; i < k+m; i++ {
		shards[i] = make([]byte, shardSize)
	}

	// Make copies of original data to check systematic preservation
	originalData := make([][]byte, k)
	for i := 0; i < k; i++ {
		originalData[i] = append([]byte(nil), shards[i]...)
	}

	if err := rs.Encode(shards); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// Verify systematic invariant
	for i := 0; i < k; i++ {
		if !bytes.Equal(shards[i], originalData[i]) {
			t.Fatalf("data shard %d was mutated during systematic encoding", i)
		}
	}

	// Verify parity validity
	ok, err := rs.Verify(shards)
	if err != nil || !ok {
		t.Fatalf("verify failed: ok=%v, err=%v", ok, err)
	}
}

func TestReedSolomonArbitraryErasureRecovery(t *testing.T) {
	testCases := []struct {
		k, m int
	}{
		{2, 2},
		{4, 4},
		{8, 8},
		{16, 16},
	}

	for _, tc := range testCases {
		k, m := tc.k, tc.m
		rs, err := reedsolomon.New(k, m)
		if err != nil {
			t.Fatalf("failed New(%d, %d): %v", k, m, err)
		}

		shardSize := 128
		shards := make([][]byte, k+m)
		originalData := make([][]byte, k)
		for i := 0; i < k; i++ {
			shards[i] = make([]byte, shardSize)
			for b := 0; b < shardSize; b++ {
				shards[i][b] = byte(rand.Intn(256))
			}
			originalData[i] = append([]byte(nil), shards[i]...)
		}
		for i := k; i < k+m; i++ {
			shards[i] = make([]byte, shardSize)
		}

		if err := rs.Encode(shards); err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		// Erase exactly m arbitrary shards (leave k shards)
		erasedShards := make([][]byte, k+m)
		for i := 0; i < k+m; i++ {
			erasedShards[i] = append([]byte(nil), shards[i]...)
		}

		// Deterministically wipe the first m/2 data shards and first m/2 parity shards
		for i := 0; i < m/2; i++ {
			erasedShards[i] = nil
			erasedShards[k+i] = nil
		}

		if err := rs.Reconstruct(erasedShards); err != nil {
			t.Fatalf("reconstruct failed for k=%d, m=%d: %v", k, m, err)
		}

		// Verify that all original data shards are reconstructed byte-for-byte
		for i := 0; i < k; i++ {
			if !bytes.Equal(erasedShards[i], originalData[i]) {
				t.Fatalf("reconstructed shard %d does not match original for k=%d, m=%d", i, k, m)
			}
		}
	}
}

func TestReedSolomonInsufficientShards(t *testing.T) {
	k, m := 4, 4
	rs, _ := reedsolomon.New(k, m)
	shards := make([][]byte, k+m)
	// Only 3 shards available (< 4)
	shards[0] = []byte{1, 2, 3, 4}
	shards[1] = []byte{5, 6, 7, 8}
	shards[2] = []byte{9, 10, 11, 12}

	err := rs.Reconstruct(shards)
	if err == nil {
		t.Fatal("expected error on insufficient shards for reconstruction")
	}
}
