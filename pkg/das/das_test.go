package das_test

import (
	"fmt"
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/das"
	"github.com/ZengLLQ/aura-da/pkg/eds"
	"github.com/ZengLLQ/aura-da/pkg/nmt"
)

func buildTestSquare(k int) (*eds.Square, *eds.DataAvailabilityHeader, error) {
	var shares []eds.Share
	for r := 0; r < k; r++ {
		for c := 0; c < k; c++ {
			ns := nmt.NewNamespaceFromUint64(uint64(r*k + c + 1))
			data := []byte(fmt.Sprintf("tx_data_payload_r%02d_c%02d_hashbytes", r, c))
			shares = append(shares, eds.NewShare(ns, data))
		}
	}

	sq, err := eds.NewSquare(k, shares)
	if err != nil {
		return nil, nil, err
	}

	dah, err := eds.ComputeDAH(sq)
	if err != nil {
		return nil, nil, err
	}

	return sq, dah, nil
}

func TestDataAvailabilitySamplingSuccess(t *testing.T) {
	k := 4 // 4x4 original -> 8x8 extended (64 shares)
	sq, dah, err := buildTestSquare(k)
	if err != nil {
		t.Fatalf("failed to build test square: %v", err)
	}

	sampleCount := 16
	samples, err := das.SampleSquare(sq, dah, sampleCount)
	if err != nil {
		t.Fatalf("SampleSquare failed: %v", err)
	}
	if len(samples) != sampleCount {
		t.Fatalf("expected %d samples, got %d", sampleCount, len(samples))
	}

	lc := das.NewLightClient(sampleCount, dah)
	ok, conf, err := lc.CheckAvailability(samples)
	if err != nil || !ok {
		t.Fatalf("CheckAvailability failed: ok=%v, conf=%f, err=%v", ok, conf, err)
	}

	// For S = 16: confidence must be > 99.99%
	if conf < 0.9999 {
		t.Fatalf("expected confidence > 0.9999, got %f", conf)
	}
}

func TestDataAvailabilitySamplingTamperDetection(t *testing.T) {
	k := 4
	sq, dah, _ := buildTestSquare(k)

	samples, _ := das.SampleSquare(sq, dah, 16)
	lc := das.NewLightClient(16, dah)

	// Tamper with one sample's payload
	samples[0].Share.Data[0] ^= 0xFF

	err := lc.VerifySamples(samples)
	if err == nil {
		t.Fatal("expected error on tampered sample payload")
	}
}
