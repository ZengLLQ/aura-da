package reedsolomon

import (
	"errors"
	"fmt"

	"github.com/ZengLLQ/aura-da/pkg/galois"
)

// Matrix represents a 2D matrix of elements over GF(2^8).
type Matrix struct {
	Rows int
	Cols int
	Data [][]byte
}

// NewMatrix creates a new Rows x Cols zero-initialized matrix.
func NewMatrix(rows, cols int) *Matrix {
	data := make([][]byte, rows)
	for i := range data {
		data[i] = make([]byte, cols)
	}
	return &Matrix{Rows: rows, Cols: cols, Data: data}
}

// IdentityMatrix returns an n x n identity matrix over GF(2^8).
func IdentityMatrix(n int) *Matrix {
	m := NewMatrix(n, n)
	for i := 0; i < n; i++ {
		m.Data[i][i] = 1
	}
	return m
}

// Multiply multiplies m by other in GF(2^8): result = m * other.
func (m *Matrix) Multiply(other *Matrix) (*Matrix, error) {
	if m.Cols != other.Rows {
		return nil, fmt.Errorf("matrix multiply: dimension mismatch (%dx%d vs %dx%d)",
			m.Rows, m.Cols, other.Rows, other.Cols)
	}
	res := NewMatrix(m.Rows, other.Cols)
	for r := 0; r < m.Rows; r++ {
		for c := 0; c < other.Cols; c++ {
			var sum byte
			for k := 0; k < m.Cols; k++ {
				sum = galois.Add(sum, galois.Mul(m.Data[r][k], other.Data[k][c]))
			}
			res.Data[r][c] = sum
		}
	}
	return res, nil
}

// Invert computes the inverse of a square matrix using Gaussian elimination over GF(2^8).
func (m *Matrix) Invert() (*Matrix, error) {
	if m.Rows != m.Cols {
		return nil, errors.New("matrix: invert requires square matrix")
	}
	n := m.Rows
	// Form augmented matrix [m | I]
	aug := NewMatrix(n, 2*n)
	for r := 0; r < n; r++ {
		copy(aug.Data[r][:n], m.Data[r])
		aug.Data[r][n+r] = 1
	}

	// Forward elimination & back-substitution
	for c := 0; c < n; c++ {
		// Find pivot row with non-zero in column c
		pivotRow := -1
		for r := c; r < n; r++ {
			if aug.Data[r][c] != 0 {
				pivotRow = r
				break
			}
		}
		if pivotRow == -1 {
			return nil, errors.New("matrix: singular matrix, cannot invert")
		}

		// Swap pivot row with row c if necessary
		if pivotRow != c {
			aug.Data[c], aug.Data[pivotRow] = aug.Data[pivotRow], aug.Data[c]
		}

		// Normalize pivot row so pivot element becomes 1
		pivotVal := aug.Data[c][c]
		pivotInv, _ := galois.Inv(pivotVal)
		for j := 0; j < 2*n; j++ {
			aug.Data[c][j] = galois.Mul(aug.Data[c][j], pivotInv)
		}

		// Eliminate all other rows in column c
		for r := 0; r < n; r++ {
			if r != c && aug.Data[r][c] != 0 {
				factor := aug.Data[r][c]
				for j := 0; j < 2*n; j++ {
					aug.Data[r][j] = galois.Sub(aug.Data[r][j], galois.Mul(factor, aug.Data[c][j]))
				}
			}
		}
	}

	// Extract the right half [I | inv]
	inv := NewMatrix(n, n)
	for r := 0; r < n; r++ {
		copy(inv.Data[r], aug.Data[r][n:])
	}
	return inv, nil
}

// SubMatrix creates a new matrix from specific rows of m.
func (m *Matrix) SubMatrix(rowIndices []int) *Matrix {
	sub := NewMatrix(len(rowIndices), m.Cols)
	for i, r := range rowIndices {
		copy(sub.Data[i], m.Data[r])
	}
	return sub
}
