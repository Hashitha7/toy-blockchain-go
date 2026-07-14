package ledger_test

import (
	"testing"

	"blockchain/block"
	"blockchain/ledger"
)

// TestOverspendRejected verifies that a transaction spending more than
// the sender's balance is rejected (FR-4 — overspending scenario).
func TestOverspendRejected(t *testing.T) {
	l := ledger.NewLedger()

	// Give Alice 100 via coinbase.
	mint := block.Transaction{From: "coinbase", To: "Alice", Amount: 100}
	if err := l.ValidateTransaction(mint); err != nil {
		t.Fatalf("coinbase txn should be valid: %v", err)
	}
	l.ApplyTransaction(mint)

	if l.Balances["Alice"] != 100 {
		t.Fatalf("Alice should have 100, got %.2f", l.Balances["Alice"])
	}

	// Try to overspend — Alice sends 150 but only has 100.
	overspend := block.Transaction{From: "Alice", To: "Bob", Amount: 150}
	err := l.ValidateTransaction(overspend)
	if err == nil {
		t.Fatal("expected overspend to be rejected, but it was accepted")
	}
	t.Logf("correctly rejected: %v", err)

	// Verify balance unchanged.
	if l.Balances["Alice"] != 100 {
		t.Errorf("Alice balance should still be 100, got %.2f", l.Balances["Alice"])
	}
	if l.Balances["Bob"] != 0 {
		t.Errorf("Bob balance should be 0, got %.2f", l.Balances["Bob"])
	}
}

// TestNegativeAmountRejected verifies that transactions with zero or
// negative amounts are rejected.
func TestNegativeAmountRejected(t *testing.T) {
	l := ledger.NewLedger()

	tests := []struct {
		name   string
		amount float64
	}{
		{"zero", 0},
		{"negative", -50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := block.Transaction{From: "Alice", To: "Bob", Amount: tt.amount}
			if err := l.ValidateTransaction(tx); err == nil {
				t.Errorf("amount %.2f should be rejected", tt.amount)
			}
		})
	}
}

// TestValidTransactionAccepted verifies that a properly funded transaction
// is accepted and balances update correctly.
func TestValidTransactionAccepted(t *testing.T) {
	l := ledger.NewLedger()

	// Mint 100 to Alice.
	l.ApplyTransaction(block.Transaction{From: "coinbase", To: "Alice", Amount: 100})

	// Alice sends 40 to Bob.
	tx := block.Transaction{From: "Alice", To: "Bob", Amount: 40}
	if err := l.ValidateTransaction(tx); err != nil {
		t.Fatalf("valid txn rejected: %v", err)
	}
	l.ApplyTransaction(tx)

	if l.Balances["Alice"] != 60 {
		t.Errorf("Alice should have 60, got %.2f", l.Balances["Alice"])
	}
	if l.Balances["Bob"] != 40 {
		t.Errorf("Bob should have 40, got %.2f", l.Balances["Bob"])
	}
}

// TestRebuildFromBlocks checks that the ledger can be reconstructed from
// a sequence of blocks.
func TestRebuildFromBlocks(t *testing.T) {
	blocks := []block.Block{
		{
			Height:       0,
			Transactions: []block.Transaction{},
		},
		{
			Height: 1,
			Transactions: []block.Transaction{
				{From: "coinbase", To: "Alice", Amount: 100},
			},
		},
		{
			Height: 2,
			Transactions: []block.Transaction{
				{From: "Alice", To: "Bob", Amount: 30},
			},
		},
	}

	l := ledger.NewLedger()
	l.RebuildFromBlocks(blocks)

	if l.Balances["Alice"] != 70 {
		t.Errorf("Alice should have 70, got %.2f", l.Balances["Alice"])
	}
	if l.Balances["Bob"] != 30 {
		t.Errorf("Bob should have 30, got %.2f", l.Balances["Bob"])
	}
}
