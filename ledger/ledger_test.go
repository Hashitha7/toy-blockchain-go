package ledger_test

import (
	"testing"

	"blockchain/block"
	"blockchain/ledger"
	"blockchain/wallet"
)

// helper to create and sign a transaction
func makeSignedTx(t *testing.T, pubKey, privKey, to string, amount int64) block.Transaction {
	tx := block.Transaction{From: pubKey, To: to, Amount: amount, PubKey: pubKey}
	sig, err := wallet.Sign(privKey, tx.SignableData())
	if err != nil {
		t.Fatalf("failed to sign tx: %v", err)
	}
	tx.Signature = sig
	return tx
}

// TestOverspendRejected verifies that a transaction spending more than
// the sender's balance is rejected (FR-4 — overspending scenario).
func TestOverspendRejected(t *testing.T) {
	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	l := ledger.NewLedger()

	// Give Alice 100 via coinbase.
	mint := block.Transaction{From: "coinbase", To: alicePub, Amount: 100}
	if err := l.ValidateTransaction(mint, nil); err != nil {
		t.Fatalf("coinbase txn should be valid: %v", err)
	}
	l.ApplyTransaction(mint)

	if l.Balances[alicePub] != 100 {
		t.Fatalf("Alice should have 100, got %d", l.Balances[alicePub])
	}

	// Try to overspend — Alice sends 150 but only has 100.
	overspend := makeSignedTx(t, alicePub, alicePriv, bobPub, 150)
	err := l.ValidateTransaction(overspend, nil)
	if err == nil {
		t.Fatal("expected overspend to be rejected, but it was accepted")
	}
	t.Logf("correctly rejected: %v", err)

	// Verify balance unchanged.
	if l.Balances[alicePub] != 100 {
		t.Errorf("Alice balance should still be 100, got %d", l.Balances[alicePub])
	}
	if l.Balances[bobPub] != 0 {
		t.Errorf("Bob balance should be 0, got %d", l.Balances[bobPub])
	}
}

// TestNegativeAmountRejected verifies that transactions with zero or
// negative amounts are rejected.
func TestNegativeAmountRejected(t *testing.T) {
	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	l := ledger.NewLedger()

	tests := []struct {
		name   string
		amount int64
	}{
		{"zero", 0},
		{"negative", -50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := makeSignedTx(t, alicePub, alicePriv, bobPub, tt.amount)
			if err := l.ValidateTransaction(tx, nil); err == nil {
				t.Errorf("amount %d should be rejected", tt.amount)
			}
		})
	}
}

// TestValidTransactionAccepted verifies that a properly funded transaction
// is accepted and balances update correctly.
func TestValidTransactionAccepted(t *testing.T) {
	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	l := ledger.NewLedger()

	// Mint 100 to Alice.
	l.ApplyTransaction(block.Transaction{From: "coinbase", To: alicePub, Amount: 100})

	// Alice sends 40 to Bob.
	tx := makeSignedTx(t, alicePub, alicePriv, bobPub, 40)
	if err := l.ValidateTransaction(tx, nil); err != nil {
		t.Fatalf("valid txn rejected: %v", err)
	}
	l.ApplyTransaction(tx)

	if l.Balances[alicePub] != 60 {
		t.Errorf("Alice should have 60, got %d", l.Balances[alicePub])
	}
	if l.Balances[bobPub] != 40 {
		t.Errorf("Bob should have 40, got %d", l.Balances[bobPub])
	}
}

// TestRebuildFromBlocks checks that the ledger can be reconstructed from
// a sequence of blocks.
func TestRebuildFromBlocks(t *testing.T) {
	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	tx := makeSignedTx(t, alicePub, alicePriv, bobPub, 30)

	blocks := []block.Block{
		{
			Height:       0,
			Transactions: []block.Transaction{},
		},
		{
			Height: 1,
			Transactions: []block.Transaction{
				{From: "coinbase", To: alicePub, Amount: 100},
			},
		},
		{
			Height: 2,
			Transactions: []block.Transaction{
				tx,
			},
		},
	}

	l := ledger.NewLedger()
	l.RebuildFromBlocks(blocks)

	if l.Balances[alicePub] != 70 {
		t.Errorf("Alice should have 70, got %d", l.Balances[alicePub])
	}
	if l.Balances[bobPub] != 30 {
		t.Errorf("Bob should have 30, got %d", l.Balances[bobPub])
	}
}

// TestPendingPoolDoubleSpendRejected verifies that already-pending spends
// are correctly subtracted from the available balance.
func TestPendingPoolDoubleSpendRejected(t *testing.T) {
	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()
	charliePub, _, _ := wallet.GenerateKeyPair()

	l := ledger.NewLedger()
	l.ApplyTransaction(block.Transaction{From: "coinbase", To: alicePub, Amount: 100})

	// Alice sends 70 to Bob (pending)
	pending := []block.Transaction{
		makeSignedTx(t, alicePub, alicePriv, bobPub, 70),
	}

	// Alice tries to send another 70 to Charlie.
	// She has 100 on chain, but 70 is already pending, so available is 30.
	tx2 := makeSignedTx(t, alicePub, alicePriv, charliePub, 70)
	err := l.ValidateTransaction(tx2, pending)
	if err == nil {
		t.Fatal("double-spend from pending pool should be rejected")
	}
	t.Logf("correctly rejected double-spend: %v", err)
}
