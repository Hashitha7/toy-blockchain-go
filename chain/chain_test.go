package chain_test

import (
	"testing"

	"blockchain/block"
	"blockchain/chain"
	"blockchain/ledger"
)

// TestValidChainPasses builds a small honest chain and confirms that
// validation reports it as valid (FR-6 — honest chain scenario).
func TestValidChainPasses(t *testing.T) {
	bc := chain.NewChain(1) // low difficulty for fast tests
	l := ledger.NewLedger()

	// Add and mine two blocks.
	tx1 := block.Transaction{From: "coinbase", To: "Alice", Amount: 100}
	bc.AddTransaction(tx1, l)
	l.ApplyTransaction(tx1) // pre-apply coinbase
	_, _, _, err := bc.MineNextBlock(l)
	if err != nil {
		t.Fatalf("mine block 1: %v", err)
	}

	tx2 := block.Transaction{From: "Alice", To: "Bob", Amount: 30}
	bc.AddTransaction(tx2, l)
	_, _, _, err = bc.MineNextBlock(l)
	if err != nil {
		t.Fatalf("mine block 2: %v", err)
	}

	result := bc.Validate()
	if !result.Valid {
		t.Errorf("expected valid chain, got: %s (block %d)", result.ErrorMessage, result.ErrorBlock)
	}
}

// TestTamperDetection modifies a transaction inside an early block and
// verifies that validation catches the tampering (FR-6 — tamper scenario).
func TestTamperDetection(t *testing.T) {
	bc := chain.NewChain(1)
	l := ledger.NewLedger()

	// Build a chain of 3 blocks.
	tx := block.Transaction{From: "coinbase", To: "Alice", Amount: 100}
	bc.AddTransaction(tx, l)
	l.ApplyTransaction(tx)
	bc.MineNextBlock(l)

	tx2 := block.Transaction{From: "Alice", To: "Bob", Amount: 20}
	bc.AddTransaction(tx2, l)
	bc.MineNextBlock(l)

	tx3 := block.Transaction{From: "Bob", To: "Charlie", Amount: 5}
	bc.AddTransaction(tx3, l)
	bc.MineNextBlock(l)

	// Tamper with block 1's transaction amount.
	bc.Blocks[1].Transactions[0].Amount = 999999

	result := bc.Validate()
	if result.Valid {
		t.Fatal("expected invalid chain after tampering, but got valid")
	}
	if result.ErrorBlock != 1 {
		t.Errorf("expected first invalid block = 1, got %d", result.ErrorBlock)
	}

	t.Logf("tamper detected: %s", result.ErrorMessage)
}

// TestGenesisOnlyChain verifies that a freshly created chain with only the
// genesis block is valid (FR-2).
func TestGenesisOnlyChain(t *testing.T) {
	bc := chain.NewChain(1)
	result := bc.Validate()
	if !result.Valid {
		t.Errorf("genesis-only chain should be valid: %s", result.ErrorMessage)
	}
	if len(bc.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(bc.Blocks))
	}
	if bc.Blocks[0].Height != 0 {
		t.Errorf("genesis height should be 0, got %d", bc.Blocks[0].Height)
	}
}
