package galois_test

import (
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/galois"
)

func TestGaloisIdentities(t *testing.T) {
	for a := 1; a < 256; a++ {
		ba := byte(a)
		// Multiplicative identity: a * 1 = a
		if galois.Mul(ba, 1) != ba {
			t.Fatalf("identity failed for %d", a)
		}

		// Inverse property: a * a^(-1) = 1
		inv, err := galois.Inv(ba)
		if err != nil {
			t.Fatalf("inverse failed for %d: %v", a, err)
		}
		if galois.Mul(ba, inv) != 1 {
			t.Fatalf("a * inv(a) != 1 for %d (inv=%d, product=%d)", a, inv, galois.Mul(ba, inv))
		}
	}
}

func TestGaloisCommutativityAndDistributivity(t *testing.T) {
	for a := 0; a < 256; a += 17 {
		for b := 0; b < 256; b += 19 {
			ba := byte(a)
			bb := byte(b)

			// Commutativity: a * b = b * a
			if galois.Mul(ba, bb) != galois.Mul(bb, ba) {
				t.Fatalf("commutativity failed for a=%d, b=%d", a, b)
			}

			// Distributivity: a * (b ^ c) = (a * b) ^ (a * c)
			for c := 0; c < 256; c += 23 {
				bc := byte(c)
				left := galois.Mul(ba, galois.Add(bb, bc))
				right := galois.Add(galois.Mul(ba, bb), galois.Mul(ba, bc))
				if left != right {
					t.Fatalf("distributivity failed for a=%d, b=%d, c=%d", a, b, c)
				}
			}
		}
	}
}

func TestGaloisDivision(t *testing.T) {
	for a := 0; a < 256; a += 11 {
		for b := 1; b < 256; b += 13 {
			ba := byte(a)
			bb := byte(b)

			q, err := galois.Div(ba, bb)
			if err != nil {
				t.Fatalf("division error for %d / %d: %v", a, b, err)
			}
			prod := galois.Mul(q, bb)
			if prod != ba {
				t.Fatalf("(a / b) * b != a for a=%d, b=%d (q=%d, prod=%d)", a, b, q, prod)
			}
		}
	}

	_, err := galois.Div(42, 0)
	if err == nil {
		t.Fatal("expected error on division by zero")
	}
}
