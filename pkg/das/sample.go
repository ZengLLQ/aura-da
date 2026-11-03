package das

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
)

// Coordinate represents a cell position (Row, Col) in the 2D Extended Data Square.
type Coordinate struct {
	Row int
	Col int
}

// Sample contains a sampled share and its dual inclusion proofs against DAH row & col roots.
type Sample struct {
	Coord    Coordinate
	Share    eds.Share
	RowProof nmt.Proof
	ColProof nmt.Proof
}

// LightClient manages Data Availability Sampling verification for resource-constrained clients.
type LightClient struct {
	SampleCount int
	DAH         *eds.DataAvailabilityHeader
}

// NewLightClient creates a light client targeting a specific sample count S.
func NewLightClient(sampleCount int, dah *eds.DataAvailabilityHeader) *LightClient {
	return &LightClient{
		SampleCount: sampleCount,
		DAH:         dah,
	}
}

// StatisticalConfidence returns the statistical probability that the block is available:
// Confidence = 1 - (1/2)^S
func (lc *LightClient) StatisticalConfidence() float64 {
	if lc.SampleCount <= 0 {
		return 0.0
	}
	probUnavailable := math.Pow(0.5, float64(lc.SampleCount))
	return 1.0 - probUnavailable
}

// SampleSquare extracts S unique random coordinates from the square with dual NMT proofs.
func SampleSquare(sq *eds.Square, dah *eds.DataAvailabilityHeader, sampleCount int) ([]Sample, error) {
	w := sq.ExtendedWidth
	totalCells := w * w
	if sampleCount <= 0 || sampleCount > totalCells {
		return nil, fmt.Errorf("das: invalid sample count %d for %dx%d square", sampleCount, w, w)
	}

	// Pre-build row and column trees to extract proofs
	rowTrees := make([]*nmt.Tree, w)
	for r := 0; r < w; r++ {
		t := nmt.New()
		for _, sh := range sq.Row(r) {
			_ = t.Push(sh.Namespace, sh.Data)
		}
		rowTrees[r] = t
	}

	colTrees := make([]*nmt.Tree, w)
	for c := 0; c < w; c++ {
		t := nmt.New()
		for _, sh := range sq.Col(c) {
			_ = t.Push(sh.Namespace, sh.Data)
		}
		colTrees[c] = t
	}

	seen := make(map[Coordinate]bool)
	var samples []Sample

	for len(samples) < sampleCount {
		rBig, _ := rand.Int(rand.Reader, big.NewInt(int64(w)))
		cBig, _ := rand.Int(rand.Reader, big.NewInt(int64(w)))
		coord := Coordinate{Row: int(rBig.Int64()), Col: int(cBig.Int64())}

		if seen[coord] {
			continue
		}
		seen[coord] = true

		sh, err := sq.GetShare(coord.Row, coord.Col)
		if err != nil {
			return nil, err
		}

		rowProof, err := rowTrees[coord.Row].ProveIndex(coord.Col)
		if err != nil {
			return nil, fmt.Errorf("das: failed row proof for (%d, %d): %w", coord.Row, coord.Col, err)
		}

		colProof, err := colTrees[coord.Col].ProveIndex(coord.Row)
		if err != nil {
			return nil, fmt.Errorf("das: failed col proof for (%d, %d): %w", coord.Row, coord.Col, err)
		}

		samples = append(samples, Sample{
			Coord:    coord,
			Share:    sh,
			RowProof: rowProof,
			ColProof: colProof,
		})
	}

	return samples, nil
}

// VerifySamples validates all sampled shares against the Data Availability Header.
func (lc *LightClient) VerifySamples(samples []Sample) error {
	if len(samples) < lc.SampleCount {
		return fmt.Errorf("das: insufficient samples provided (got %d, want %d)",
			len(samples), lc.SampleCount)
	}

	w := len(lc.DAH.RowRoots)

	for i, sample := range samples {
		r := sample.Coord.Row
		c := sample.Coord.Col

		if r < 0 || r >= w || c < 0 || c >= w {
			return fmt.Errorf("das: sample %d coordinate (%d, %d) out of bounds", i, r, c)
		}

		// Verify against row root
		rowRoot := lc.DAH.RowRoots[r]
		if !nmt.VerifyLeafIndex(rowRoot, sample.Share.Namespace, sample.Share.Data, sample.RowProof) {
			return fmt.Errorf("das: sample %d failed row root verification at (%d, %d)", i, r, c)
		}

		// Verify against col root
		colRoot := lc.DAH.ColRoots[c]
		if !nmt.VerifyLeafIndex(colRoot, sample.Share.Namespace, sample.Share.Data, sample.ColProof) {
			return fmt.Errorf("das: sample %d failed col root verification at (%d, %d)", i, r, c)
		}
	}

	return nil
}

// CheckAvailability performs end-to-end verification and returns confidence metrics.
func (lc *LightClient) CheckAvailability(samples []Sample) (bool, float64, error) {
	if err := lc.VerifySamples(samples); err != nil {
		return false, 0.0, err
	}
	confidence := lc.StatisticalConfidence()
	if confidence < 0.99 {
		return false, confidence, errors.New("das: statistical confidence below required threshold (99%)")
	}
	return true, confidence, nil
}
