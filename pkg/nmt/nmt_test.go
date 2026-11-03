package nmt_test

import (
	"testing"

	"github.com/ZengLLQ/aura-da/pkg/nmt"
)

func TestNMTRootCalculation(t *testing.T) {
	tree := nmt.New()
	ns1 := nmt.NewNamespaceFromUint64(1)
	ns2 := nmt.NewNamespaceFromUint64(2)

	_ = tree.Push(ns1, []byte("data_1"))
	_ = tree.Push(ns2, []byte("data_2"))

	root, err := tree.Root()
	if err != nil {
		t.Fatalf("failed to calculate root: %v", err)
	}

	if !root.MinNS.Equals(ns1) {
		t.Fatalf("root MinNS expected %x, got %x", ns1[:], root.MinNS[:])
	}
	if !root.MaxNS.Equals(ns2) {
		t.Fatalf("root MaxNS expected %x, got %x", ns2[:], root.MaxNS[:])
	}
}

func TestNMTOutOfOrderRejection(t *testing.T) {
	tree := nmt.New()
	nsHigh := nmt.NewNamespaceFromUint64(10)
	nsLow := nmt.NewNamespaceFromUint64(5)

	_ = tree.Push(nsHigh, []byte("high"))
	err := tree.Push(nsLow, []byte("low"))
	if err == nil {
		t.Fatal("expected error on out-of-order namespace push")
	}
}

func TestNMTInclusionProof(t *testing.T) {
	tree := nmt.New()
	ns0 := nmt.NewNamespaceFromUint64(0)
	ns1 := nmt.NewNamespaceFromUint64(1)
	ns2 := nmt.NewNamespaceFromUint64(2)
	ns3 := nmt.NewNamespaceFromUint64(3)

	_ = tree.Push(ns0, []byte("leaf_0"))
	_ = tree.Push(ns1, []byte("leaf_1a"))
	_ = tree.Push(ns1, []byte("leaf_1b"))
	_ = tree.Push(ns2, []byte("leaf_2"))
	_ = tree.Push(ns3, []byte("leaf_3"))

	root, err := tree.Root()
	if err != nil {
		t.Fatalf("root error: %v", err)
	}

	// 1. Prove namespace 1 (contains 2 leaves)
	proof1, err := tree.ProveNamespace(ns1)
	if err != nil {
		t.Fatalf("prove namespace 1 failed: %v", err)
	}
	leavesData := [][]byte{[]byte("leaf_1a"), []byte("leaf_1b")}
	if !nmt.VerifyInclusion(root, ns1, leavesData, proof1) {
		t.Fatal("inclusion proof verification failed for ns1")
	}

	// 2. Tampered data should fail
	tamperedData := [][]byte{[]byte("leaf_1a_tampered"), []byte("leaf_1b")}
	if nmt.VerifyInclusion(root, ns1, tamperedData, proof1) {
		t.Fatal("verification should fail on tampered leaf data")
	}

	// 3. Prove single index
	proofIdx3, err := tree.ProveIndex(3)
	if err != nil {
		t.Fatalf("prove index 3 failed: %v", err)
	}
	if !nmt.VerifyLeafIndex(root, ns2, []byte("leaf_2"), proofIdx3) {
		t.Fatal("leaf index verification failed for idx 3")
	}
}

func TestNMTAbsenceProof(t *testing.T) {
	tree := nmt.New()
	ns1 := nmt.NewNamespaceFromUint64(1)
	ns3 := nmt.NewNamespaceFromUint64(3)

	_ = tree.Push(ns1, []byte("item_1"))
	_ = tree.Push(ns3, []byte("item_3"))

	root, _ := tree.Root()

	// Prove absence of namespace 2
	ns2 := nmt.NewNamespaceFromUint64(2)
	proofAbsence, err := tree.ProveNamespace(ns2)
	if err != nil {
		t.Fatalf("failed to generate absence proof: %v", err)
	}
	if !proofAbsence.IsAbsence {
		t.Fatal("expected IsAbsence to be true")
	}

	// Verify absence against adjacent leaf (item_3 with ns3)
	if !nmt.VerifyAbsence(root, ns2, []byte("item_3"), ns3, proofAbsence) {
		t.Fatal("absence verification failed for ns2")
	}
}
