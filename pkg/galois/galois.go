package galois

import "errors"

// Primitive polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x11D = 285)
const (
	FieldSize = 256
	Modulus   = 285
)

var (
	expTable [512]byte
	logTable [FieldSize]byte
)

func init() {
	var x byte = 1
	for i := 0; i < 255; i++ {
		expTable[i] = x
		expTable[i+255] = x
		logTable[x] = byte(i)

		// Multiply by generator alpha = 2 (shift left by 1)
		hi := x & 0x80
		x <<= 1
		if hi != 0 {
			x ^= (Modulus & 0xFF)
		}
	}
	// log(0) is mathematically undefined; logTable[0] remains 0.
}

// Add computes a + b in GF(2^8) (equivalent to bitwise XOR).
func Add(a, b byte) byte {
	return a ^ b
}

// Sub computes a - b in GF(2^8) (equivalent to bitwise XOR).
func Sub(a, b byte) byte {
	return a ^ b
}

// Mul computes a * b in GF(2^8) using logarithm lookup tables.
func Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	logA := int(logTable[a])
	logB := int(logTable[b])
	return expTable[logA+logB]
}

// Div computes a / b in GF(2^8).
func Div(a, b byte) (byte, error) {
	if b == 0 {
		return 0, errors.New("galois: division by zero")
	}
	if a == 0 {
		return 0, nil
	}
	logA := int(logTable[a])
	logB := int(logTable[b])
	diff := (logA - logB + 255) % 255
	return expTable[diff], nil
}

// Inv computes the multiplicative inverse a^(-1) in GF(2^8).
func Inv(a byte) (byte, error) {
	if a == 0 {
		return 0, errors.New("galois: zero has no multiplicative inverse")
	}
	logA := int(logTable[a])
	diff := (255 - logA) % 255
	return expTable[diff], nil
}

// MulSlice multiplies all elements in slice data by scalar in GF(2^8).
func MulSlice(scalar byte, data []byte) {
	if scalar == 0 {
		for i := range data {
			data[i] = 0
		}
		return
	}
	if scalar == 1 {
		return
	}
	logS := int(logTable[scalar])
	for i, v := range data {
		if v != 0 {
			data[i] = expTable[logS+int(logTable[v])]
		}
	}
}

// AddSlice adds src to dst element-wise in GF(2^8) (dst ^= src).
func AddSlice(dst, src []byte) {
	for i := 0; i < len(dst) && i < len(src); i++ {
		dst[i] ^= src[i]
	}
}
