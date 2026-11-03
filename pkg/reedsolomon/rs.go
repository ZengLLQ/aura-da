package reedsolomon

import (
	"errors"
	"fmt"

	"github.com/ZengLLQ/aura-da/pkg/galois"
)

// ReedSolomon implements systematic Cauchy Reed-Solomon erasure coding.
type ReedSolomon struct {
	DataShards   int
	ParityShards int
	TotalShards  int
	Generator    *Matrix
}

// New creates a new ReedSolomon encoder/decoder with dataShards and parityShards.
func New(dataShards, parityShards int) (*ReedSolomon, error) {
	if dataShards <= 0 || parityShards <= 0 {
		return nil, errors.New("reedsolomon: shard counts must be strictly positive")
	}
	total := dataShards + parityShards
	if total > 256 {
		return nil, errors.New("reedsolomon: total shards cannot exceed GF(2^8) field size (256)")
	}

	// Build Cauchy-based systematic generator matrix:
	// Top dataShards rows: Identity matrix I_k
	// Bottom parityShards rows: Cauchy matrix A_ij = 1 / (X_i ^ Y_j)
	gen := NewMatrix(total, dataShards)

	// Fill top identity submatrix
	for i := 0; i < dataShards; i++ {
		gen.Data[i][i] = 1
	}

	// Disjoint sets X and Y for Cauchy matrix:
	// X_i = dataShards + i, Y_j = j
	for i := 0; i < parityShards; i++ {
		xi := byte(dataShards + i)
		for j := 0; j < dataShards; j++ {
			yj := byte(j)
			diff := galois.Add(xi, yj) // xi ^ yj != 0 since X and Y are disjoint
			inv, err := galois.Inv(diff)
			if err != nil {
				return nil, fmt.Errorf("reedsolomon: failed computing Cauchy generator element: %w", err)
			}
			gen.Data[dataShards+i][j] = inv
		}
	}

	return &ReedSolomon{
		DataShards:   dataShards,
		ParityShards: parityShards,
		TotalShards:  total,
		Generator:    gen,
	}, nil
}

// Encode takes shards of length TotalShards (where shards[0..DataShards-1] contain data,
// and shards[DataShards..TotalShards-1] are pre-allocated buffers) and computes parity shards.
func (rs *ReedSolomon) Encode(shards [][]byte) error {
	if len(shards) != rs.TotalShards {
		return fmt.Errorf("reedsolomon: expected %d shards, got %d", rs.TotalShards, len(shards))
	}
	shardSize := len(shards[0])
	if shardSize == 0 {
		return errors.New("reedsolomon: shard size cannot be zero")
	}
	for i := 1; i < rs.TotalShards; i++ {
		if len(shards[i]) != shardSize {
			return errors.New("reedsolomon: all shards must have identical lengths")
		}
	}

	// Compute each parity shard
	for p := 0; p < rs.ParityShards; p++ {
		parityIdx := rs.DataShards + p
		target := shards[parityIdx]
		// Clear parity buffer
		for b := range target {
			target[b] = 0
		}
		// target = sum_{d=0}^{DataShards-1} Generator[parityIdx][d] * shards[d]
		for d := 0; d < rs.DataShards; d++ {
			coeff := rs.Generator.Data[parityIdx][d]
			if coeff == 0 {
				continue
			}
			for b := 0; b < shardSize; b++ {
				term := galois.Mul(coeff, shards[d][b])
				target[b] = galois.Add(target[b], term)
			}
		}
	}

	return nil
}

// Reconstruct recovers missing shards in shards (represented by nil or empty buffers).
// Requires at least DataShards valid shards to succeed.
func (rs *ReedSolomon) Reconstruct(shards [][]byte) error {
	if len(shards) != rs.TotalShards {
		return fmt.Errorf("reedsolomon: expected %d shards, got %d", rs.TotalShards, len(shards))
	}

	var presentIndices []int
	var shardSize int

	for i, s := range shards {
		if len(s) > 0 {
			presentIndices = append(presentIndices, i)
			if shardSize == 0 {
				shardSize = len(s)
			}
		}
	}

	if len(presentIndices) < rs.DataShards {
		return fmt.Errorf("reedsolomon: insufficient shards to reconstruct (need %d, have %d)",
			rs.DataShards, len(presentIndices))
	}

	// Allocate any missing shards
	for i := range shards {
		if len(shards[i]) == 0 {
			shards[i] = make([]byte, shardSize)
		}
	}

	// If all shards are present, no recovery needed
	if len(presentIndices) == rs.TotalShards {
		return nil
	}

	// Select the first DataShards available indices
	selectedIndices := presentIndices[:rs.DataShards]
	subGen := rs.Generator.SubMatrix(selectedIndices)

	// Invert the submatrix
	invSubGen, err := subGen.Invert()
	if err != nil {
		return fmt.Errorf("reedsolomon: failed to invert submatrix: %w", err)
	}

	// Recover original data shards: Data = invSubGen * AvailableShards
	recoveredData := make([][]byte, rs.DataShards)
	for d := 0; d < rs.DataShards; d++ {
		recoveredData[d] = make([]byte, shardSize)
		for i, origIdx := range selectedIndices {
			coeff := invSubGen.Data[d][i]
			if coeff == 0 {
				continue
			}
			src := shards[origIdx]
			for b := 0; b < shardSize; b++ {
				term := galois.Mul(coeff, src[b])
				recoveredData[d][b] = galois.Add(recoveredData[d][b], term)
			}
		}
	}

	// Copy recovered data back into data positions
	for d := 0; d < rs.DataShards; d++ {
		copy(shards[d], recoveredData[d])
	}

	// Re-encode all parity shards to restore full codeword
	return rs.Encode(shards)
}

// Verify checks whether all parity shards match the encoded data shards.
func (rs *ReedSolomon) Verify(shards [][]byte) (bool, error) {
	if len(shards) != rs.TotalShards {
		return false, fmt.Errorf("reedsolomon: expected %d shards, got %d", rs.TotalShards, len(shards))
	}
	shardSize := len(shards[0])

	tempShards := make([][]byte, rs.TotalShards)
	for i := 0; i < rs.DataShards; i++ {
		tempShards[i] = shards[i]
	}
	for i := rs.DataShards; i < rs.TotalShards; i++ {
		tempShards[i] = make([]byte, shardSize)
	}

	if err := rs.Encode(tempShards); err != nil {
		return false, err
	}

	for p := rs.DataShards; p < rs.TotalShards; p++ {
		for b := 0; b < shardSize; b++ {
			if shards[p][b] != tempShards[p][b] {
				return false, nil
			}
		}
	}
	return true, nil
}
