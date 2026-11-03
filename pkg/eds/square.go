package eds

import (
	"errors"
	"fmt"

	"github.com/ZengLLQ/aura-da/pkg/nmt"
	"github.com/ZengLLQ/aura-da/pkg/reedsolomon"
)

// Square represents a 2D Extended Data Square (EDS) of size 2k x 2k.
type Square struct {
	OriginalWidth int       // k
	ExtendedWidth int       // 2k
	ShareSize     int       // byte length of each share
	Shares        [][]Share // 2k x 2k grid of shares
}

// NewSquare constructs and extends a 2k x 2k EDS from k x k original shares.
func NewSquare(k int, originalShares []Share) (*Square, error) {
	if k <= 0 {
		return nil, errors.New("eds: original width k must be > 0")
	}
	expectedOriginal := k * k
	if len(originalShares) != expectedOriginal {
		return nil, fmt.Errorf("eds: expected %d original shares, got %d", expectedOriginal, len(originalShares))
	}

	shareSize := len(originalShares[0].Data)
	for i, s := range originalShares {
		if len(s.Data) != shareSize {
			return nil, fmt.Errorf("eds: share %d has irregular byte size", i)
		}
	}

	extendedWidth := 2 * k
	rs, err := reedsolomon.New(k, k)
	if err != nil {
		return nil, fmt.Errorf("eds: failed to init ReedSolomon: %w", err)
	}

	// Initialize 2k x 2k grid
	grid := make([][]Share, extendedWidth)
	for r := range grid {
		grid[r] = make([]Share, extendedWidth)
	}

	// 1. Populate Quadrant 1 (Q1: top-left k x k)
	for r := 0; r < k; r++ {
		for c := 0; c < k; c++ {
			grid[r][c] = originalShares[r*k+c]
		}
	}

	// 2. Extend each row horizontally across Q1 -> Q2 (top-right k x k)
	// Parity shares are assigned ParityNamespace (0xFF..FF)
	for r := 0; r < k; r++ {
		rowData := make([][]byte, extendedWidth)
		for c := 0; c < k; c++ {
			rowData[c] = grid[r][c].Data
		}
		for c := k; c < extendedWidth; c++ {
			rowData[c] = make([]byte, shareSize)
		}

		if err := rs.Encode(rowData); err != nil {
			return nil, fmt.Errorf("eds: failed encoding row %d: %w", r, err)
		}

		for c := k; c < extendedWidth; c++ {
			grid[r][c] = NewShare(nmt.MaxNamespace(), rowData[c])
		}
	}

	// 3. Extend each column vertically across (Q1+Q2) -> (Q3+Q4)
	for c := 0; c < extendedWidth; c++ {
		colData := make([][]byte, extendedWidth)
		for r := 0; r < k; r++ {
			colData[r] = grid[r][c].Data
		}
		for r := k; r < extendedWidth; r++ {
			colData[r] = make([]byte, shareSize)
		}

		if err := rs.Encode(colData); err != nil {
			return nil, fmt.Errorf("eds: failed encoding column %d: %w", c, err)
		}

		for r := k; r < extendedWidth; r++ {
			grid[r][c] = NewShare(nmt.MaxNamespace(), colData[r])
		}
	}

	return &Square{
		OriginalWidth: k,
		ExtendedWidth: extendedWidth,
		ShareSize:     shareSize,
		Shares:        grid,
	}, nil
}

// Row returns all shares in row r.
func (sq *Square) Row(r int) []Share {
	out := make([]Share, sq.ExtendedWidth)
	copy(out, sq.Shares[r])
	return out
}

// Col returns all shares in column c.
func (sq *Square) Col(c int) []Share {
	out := make([]Share, sq.ExtendedWidth)
	for r := 0; r < sq.ExtendedWidth; r++ {
		out[r] = sq.Shares[r][c]
	}
	return out
}

// GetShare returns the share at coordinate (r, c).
func (sq *Square) GetShare(r, c int) (Share, error) {
	if r < 0 || r >= sq.ExtendedWidth || c < 0 || c >= sq.ExtendedWidth {
		return Share{}, fmt.Errorf("eds: coordinate (%d, %d) out of bounds", r, c)
	}
	return sq.Shares[r][c], nil
}
