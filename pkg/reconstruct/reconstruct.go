package reconstruct

import (
	"errors"
	"fmt"

	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
	"github.com/ZengLLQ/aura-da/pkg/reedsolomon"
)

// PartiallyAvailableSquare represents a 2k x 2k grid with missing shares (nil).
type PartiallyAvailableSquare struct {
	OriginalWidth int
	ExtendedWidth int
	ShareSize     int
	Grid          [][]*eds.Share
}

// NewPartiallyAvailableSquare initializes an empty 2k x 2k grid.
func NewPartiallyAvailableSquare(k int, shareSize int) *PartiallyAvailableSquare {
	w := 2 * k
	grid := make([][]*eds.Share, w)
	for r := range grid {
		grid[r] = make([]*eds.Share, w)
	}
	return &PartiallyAvailableSquare{
		OriginalWidth: k,
		ExtendedWidth: w,
		ShareSize:     shareSize,
		Grid:          grid,
	}
}

// SetShare records an available share at coordinate (r, c).
func (pas *PartiallyAvailableSquare) SetShare(r, c int, sh eds.Share) {
	s := sh
	pas.Grid[r][c] = &s
}

// AvailableCount returns the count of non-nil shares currently stored.
func (pas *PartiallyAvailableSquare) AvailableCount() int {
	cnt := 0
	for r := 0; r < pas.ExtendedWidth; r++ {
		for c := 0; c < pas.ExtendedWidth; c++ {
			if pas.Grid[r][c] != nil {
				cnt++
			}
		}
	}
	return cnt
}

// ReconstructCascading iteratively decodes rows and columns until the entire square is restored.
func (pas *PartiallyAvailableSquare) ReconstructCascading() (*eds.Square, error) {
	k := pas.OriginalWidth
	w := pas.ExtendedWidth
	totalCells := w * w

	rs, err := reedsolomon.New(k, k)
	if err != nil {
		return nil, fmt.Errorf("reconstruct: failed to init ReedSolomon: %w", err)
	}

	for {
		progress := false

		// 1. Row decoding pass
		for r := 0; r < w; r++ {
			var presentCount int
			for c := 0; c < w; c++ {
				if pas.Grid[r][c] != nil {
					presentCount++
				}
			}

			// If row is not completely filled, but has >= k shares:
			if presentCount < w && presentCount >= k {
				rowShards := make([][]byte, w)
				for c := 0; c < w; c++ {
					if pas.Grid[r][c] != nil {
						rowShards[c] = pas.Grid[r][c].Data
					}
				}

				if err := rs.Reconstruct(rowShards); err != nil {
					return nil, fmt.Errorf("reconstruct: row %d decode error: %w", r, err)
				}

				// Fill missing shares
				for c := 0; c < w; c++ {
					if pas.Grid[r][c] == nil {
						var ns nmt.NamespaceID
						if r < k && c < k {
							// In practical DA, data namespaces are carried in headers/metadata
							ns = nmt.NewNamespaceFromUint64(uint64(r*k + c + 1))
						} else {
							ns = nmt.MaxNamespace()
						}
						sh := eds.NewShare(ns, rowShards[c])
						pas.Grid[r][c] = &sh
						progress = true
					}
				}
			}
		}

		// 2. Column decoding pass
		for c := 0; c < w; c++ {
			var presentCount int
			for r := 0; r < w; r++ {
				if pas.Grid[r][c] != nil {
					presentCount++
				}
			}

			if presentCount < w && presentCount >= k {
				colShards := make([][]byte, w)
				for r := 0; r < w; r++ {
					if pas.Grid[r][c] != nil {
						colShards[r] = pas.Grid[r][c].Data
					}
				}

				if err := rs.Reconstruct(colShards); err != nil {
					return nil, fmt.Errorf("reconstruct: col %d decode error: %w", c, err)
				}

				for r := 0; r < w; r++ {
					if pas.Grid[r][c] == nil {
						var ns nmt.NamespaceID
						if r < k && c < k {
							ns = nmt.NewNamespaceFromUint64(uint64(r*k + c + 1))
						} else {
							ns = nmt.MaxNamespace()
						}
						sh := eds.NewShare(ns, colShards[r])
						pas.Grid[r][c] = &sh
						progress = true
					}
				}
			}
		}

		// Check if fully reconstructed
		if pas.AvailableCount() == totalCells {
			break
		}

		// If no new shares were unlocked in this round, cannot proceed further
		if !progress {
			return nil, fmt.Errorf("reconstruct: cascading halted with %d / %d shares recovered",
				pas.AvailableCount(), totalCells)
		}
	}

	// Build full Square struct
	fullShares := make([][]eds.Share, w)
	for r := 0; r < w; r++ {
		fullShares[r] = make([]eds.Share, w)
		for c := 0; c < w; c++ {
			if pas.Grid[r][c] == nil {
				return nil, errors.New("reconstruct: missing share in final square assembly")
			}
			fullShares[r][c] = *pas.Grid[r][c]
		}
	}

	return &eds.Square{
		OriginalWidth: k,
		ExtendedWidth: w,
		ShareSize:     pas.ShareSize,
		Shares:        fullShares,
	}, nil
}
