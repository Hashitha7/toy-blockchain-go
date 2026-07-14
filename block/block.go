// Package block defines the Block data structure, deterministic SHA-256
// hashing, and proof-of-work mining for the toy blockchain.
package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Transaction represents a transfer of funds between two accounts.
// From is the sender, To is the recipient, and Amount is the value transferred.
// A special sender value "coinbase" is used to mint new funds into the system.
type Transaction struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

// Block represents a single block in the blockchain.
// Each block contains a height (index), a Unix timestamp, a list of
// transactions, the hash of the previous block, a nonce used for
// proof-of-work mining, and its own computed hash.
type Block struct {
	Height       int           `json:"height"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	PrevHash     string        `json:"prev_hash"`
	Nonce        int           `json:"nonce"`
	Hash         string        `json:"hash"`
}

// hashInput is a helper struct used to build a stable, deterministic
// serialisation of block fields for hashing. The Hash field of the block
// is deliberately excluded so that we can compute (and later verify) the
// hash from the remaining fields.
type hashInput struct {
	Height       int           `json:"height"`
	Timestamp    int64         `json:"timestamp"`
	PrevHash     string        `json:"prev_hash"`
	Nonce        int           `json:"nonce"`
	Transactions []Transaction `json:"transactions"`
}

// ComputeHash calculates the SHA-256 hash of the block based on a
// deterministic serialisation of its fields (excluding the Hash field itself).
//
// Field order fed into the hash:
//
//	Height → Timestamp → PrevHash → Nonce → Transactions (JSON array)
//
// encoding/json produces deterministic output for the same Go struct values,
// so hashing the same block twice always yields the same digest.
func ComputeHash(b Block) string {
	input := hashInput{
		Height:       b.Height,
		Timestamp:    b.Timestamp,
		PrevHash:     b.PrevHash,
		Nonce:        b.Nonce,
		Transactions: b.Transactions,
	}

	data, err := json.Marshal(input)
	if err != nil {
		// This should never happen with simple struct types.
		panic(fmt.Sprintf("block: failed to marshal hash input: %v", err))
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// GenesisBlock creates the deterministic genesis block at height 0.
// Its previous-hash is the well-known value of 64 hex zeroes, and it
// carries no transactions. The timestamp is fixed at Unix epoch 0 so that
// every fresh chain starts identically.
func GenesisBlock() Block {
	b := Block{
		Height:       0,
		Timestamp:    0,
		Transactions: []Transaction{},
		PrevHash:     strings.Repeat("0", 64),
		Nonce:        0,
	}
	b.Hash = ComputeHash(b)
	return b
}

// MineBlock creates a new block at the given height, linked to prevHash,
// containing the supplied transactions. It searches for a nonce such that
// the block's hash begins with `difficulty` leading hex zeroes.
//
// Returns the mined block, the number of hashes attempted, and the wall-clock
// duration spent mining.
func MineBlock(height int, prevHash string, txns []Transaction, difficulty int) (Block, int, time.Duration) {
	prefix := strings.Repeat("0", difficulty)

	b := Block{
		Height:       height,
		Timestamp:    time.Now().Unix(),
		Transactions: txns,
		PrevHash:     prevHash,
		Nonce:        0,
	}

	start := time.Now()
	attempts := 0

	for {
		b.Hash = ComputeHash(b)
		attempts++
		if strings.HasPrefix(b.Hash, prefix) {
			break
		}
		b.Nonce++
	}

	return b, attempts, time.Since(start)
}
