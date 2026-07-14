package block_test

import (
	"strings"
	"testing"

	"blockchain/block"
)

// TestComputeHashDeterministic verifies that hashing the same block twice
// always produces the same result (FR-3).
func TestComputeHashDeterministic(t *testing.T) {
	b := block.Block{
		Height:    1,
		Timestamp: 1700000000,
		Transactions: []block.Transaction{
			{From: "Alice", To: "Bob", Amount: 50},
		},
		PrevHash: strings.Repeat("0", 64),
		Nonce:    42,
	}
	b.MerkleRoot = block.ComputeMerkleRoot(b.Transactions)

	hash1 := block.ComputeHash(b)
	hash2 := block.ComputeHash(b)

	if hash1 != hash2 {
		t.Errorf("hash is not deterministic: %s != %s", hash1, hash2)
	}

	if len(hash1) != 64 {
		t.Errorf("expected 64-char hex hash, got length %d", len(hash1))
	}
}

// TestComputeHashChangesWithFields ensures that changing any field produces
// a different hash.
func TestComputeHashChangesWithFields(t *testing.T) {
	base := block.Block{
		Height:       1,
		Timestamp:    1700000000,
		Transactions: []block.Transaction{{From: "A", To: "B", Amount: 10}},
		PrevHash:     strings.Repeat("0", 64),
		Nonce:        0,
	}
	base.MerkleRoot = block.ComputeMerkleRoot(base.Transactions)

	baseHash := block.ComputeHash(base)

	// Change nonce
	modified := base
	modified.Nonce = 1
	if block.ComputeHash(modified) == baseHash {
		t.Error("changing nonce should change hash")
	}

	// Change height
	modified = base
	modified.Height = 2
	if block.ComputeHash(modified) == baseHash {
		t.Error("changing height should change hash")
	}

	// Change timestamp
	modified = base
	modified.Timestamp = 9999999999
	if block.ComputeHash(modified) == baseHash {
		t.Error("changing timestamp should change hash")
	}

	// Change transaction
	modified = base
	modified.Transactions = []block.Transaction{{From: "A", To: "B", Amount: 20}}
	modified.MerkleRoot = block.ComputeMerkleRoot(modified.Transactions)
	if block.ComputeHash(modified) == baseHash {
		t.Error("changing transaction amount should change hash")
	}
}

// TestGenesisBlock verifies the genesis block properties (FR-2).
func TestGenesisBlock(t *testing.T) {
	g := block.GenesisBlock()

	if g.Height != 0 {
		t.Errorf("genesis height = %d, want 0", g.Height)
	}
	if g.PrevHash != strings.Repeat("0", 64) {
		t.Errorf("genesis prev_hash should be 64 zeros")
	}
	if len(g.Transactions) != 0 {
		t.Errorf("genesis should have 0 transactions, got %d", len(g.Transactions))
	}
	if g.Hash == "" {
		t.Error("genesis hash should not be empty")
	}

	// Verify stored hash matches recomputed hash
	recomputed := block.ComputeHash(g)
	if g.Hash != recomputed {
		t.Errorf("genesis hash mismatch: stored %s, recomputed %s", g.Hash, recomputed)
	}
}

// TestMineBlockMeetsDifficulty verifies that a mined block's hash starts
// with the required number of leading zeros (FR-5).
func TestMineBlockMeetsDifficulty(t *testing.T) {
	txns := []block.Transaction{
		{From: "coinbase", To: "Alice", Amount: 100},
	}

	for _, diff := range []int{1, 2, 3} {
		t.Run(strings.Repeat("0", diff), func(t *testing.T) {
			mined, attempts, elapsed := block.MineBlock(1, strings.Repeat("0", 64), txns, diff)

			prefix := strings.Repeat("0", diff)
			if !strings.HasPrefix(mined.Hash, prefix) {
				t.Errorf("difficulty %d: hash %s doesn't start with %s", diff, mined.Hash, prefix)
			}

			// Verify the nonce reproduces the exact hash
			recomputed := block.ComputeHash(mined)
			if mined.Hash != recomputed {
				t.Errorf("mined hash doesn't match recomputed hash")
			}

			t.Logf("difficulty=%d attempts=%d elapsed=%s hash=%s", diff, attempts, elapsed, mined.Hash[:16]+"...")
		})
	}
}

// TestComputeMerkleRoot validates that the Merkle tree hashes correctly.
func TestComputeMerkleRoot(t *testing.T) {
	txns := []block.Transaction{
		{From: "A", To: "B", Amount: 10},
		{From: "C", To: "D", Amount: 20},
		{From: "E", To: "F", Amount: 30},
	}
	root := block.ComputeMerkleRoot(txns)
	if len(root) != 64 {
		t.Errorf("expected 64 character hex string, got %d", len(root))
	}
}
