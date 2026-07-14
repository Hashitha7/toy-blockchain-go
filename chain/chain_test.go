package chain_test

import (
	"testing"

	"blockchain/block"
	"blockchain/chain"
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

// TestValidChainPasses builds a small honest chain and confirms that
// validation reports it as valid (FR-6 — honest chain scenario).
func TestValidChainPasses(t *testing.T) {
	bc := chain.NewChain(1) // low difficulty for fast tests
	l := ledger.NewLedger()

	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	// Add and mine two blocks.
	tx1 := block.Transaction{From: "coinbase", To: alicePub, Amount: 100}
	bc.AddTransaction(tx1, l)
	_, _, _, err := bc.MineNextBlock(l)
	if err != nil {
		t.Fatalf("mine block 1: %v", err)
	}

	tx2 := makeSignedTx(t, alicePub, alicePriv, bobPub, 30)
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

	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, bobPriv, _ := wallet.GenerateKeyPair()
	charliePub, _, _ := wallet.GenerateKeyPair()

	// Build a chain of 3 blocks.
	tx := block.Transaction{From: "coinbase", To: alicePub, Amount: 100}
	bc.AddTransaction(tx, l)
	bc.MineNextBlock(l)

	tx2 := makeSignedTx(t, alicePub, alicePriv, bobPub, 20)
	bc.AddTransaction(tx2, l)
	bc.MineNextBlock(l)

	tx3 := makeSignedTx(t, bobPub, bobPriv, charliePub, 5)
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

// TestMinedOverspendValidationFailed verifies that if an invalid transaction
// (e.g. an overspend) somehow makes it into a block on disk, the Validate()
// function catches it during ledger replay.
func TestMinedOverspendValidationFailed(t *testing.T) {
	bc := chain.NewChain(1)
	l := ledger.NewLedger()

	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	tx := block.Transaction{From: "coinbase", To: alicePub, Amount: 100}
	bc.AddTransaction(tx, l)
	bc.MineNextBlock(l)

	// Manually construct a block with an overspend to bypass pending pool checks
	overspendTx := makeSignedTx(t, alicePub, alicePriv, bobPub, 150)
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	// Create and mine the invalid block
	invalidBlock, _, _ := block.MineBlock(lastBlock.Height+1, lastBlock.Hash, []block.Transaction{overspendTx}, bc.Difficulty)
	bc.Blocks = append(bc.Blocks, invalidBlock)

	result := bc.Validate()
	if result.Valid {
		t.Fatal("expected chain to be invalid due to mined overspend, but it was valid")
	}
	if result.ErrorBlock != 2 {
		t.Errorf("expected error at block 2, got %d", result.ErrorBlock)
	}
	t.Logf("correctly caught mined overspend: %s", result.ErrorMessage)
}

// TestTamperedAndReminedBlockDetected verifies that if an attacker modifies a block
// AND re-mines it so its hash and PoW are valid, the chain still fails validation
// because the next block's prev_hash link is broken.
func TestTamperedAndReminedBlockDetected(t *testing.T) {
	bc := chain.NewChain(1)
	l := ledger.NewLedger()

	alicePub, alicePriv, _ := wallet.GenerateKeyPair()
	bobPub, _, _ := wallet.GenerateKeyPair()

	// Block 1
	tx1 := block.Transaction{From: "coinbase", To: alicePub, Amount: 100}
	bc.AddTransaction(tx1, l)
	bc.MineNextBlock(l)

	// Block 2
	tx2 := makeSignedTx(t, alicePub, alicePriv, bobPub, 20)
	bc.AddTransaction(tx2, l)
	bc.MineNextBlock(l)

	// Tamper with Block 1
	bc.Blocks[1].Transactions[0].Amount = 999999

	// Re-mine Block 1 to fix its own hash and PoW
	remined, _, _ := block.MineBlock(1, bc.Blocks[0].Hash, bc.Blocks[1].Transactions, bc.Difficulty)
	bc.Blocks[1] = remined

	// Validate the chain - should fail at Block 2 because its prev_hash points to the old Block 1
	result := bc.Validate()
	if result.Valid {
		t.Fatal("expected chain to be invalid due to broken prev-hash link, but it was valid")
	}
	if result.ErrorBlock != 2 {
		t.Errorf("expected error at block 2, got %d", result.ErrorBlock)
	}
	t.Logf("correctly caught re-mined tamper via prev-hash link: %s", result.ErrorMessage)
}
