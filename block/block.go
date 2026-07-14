// Package block defines the Block data structure, deterministic SHA-256
// hashing, and proof-of-work mining for the toy blockchain.
package block

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// Transaction represents a transfer of funds between two accounts.
// From is the sender, To is the recipient, and Amount is the value transferred.
// A special sender value "coinbase" is used to mint new funds into the system.
type Transaction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int64  `json:"amount"`
	PubKey    string `json:"pub_key,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// SignableData returns a deterministic byte representation of the transaction
// for signing and verification.
func (tx Transaction) SignableData() []byte {
	return []byte(fmt.Sprintf("%s:%s:%d", tx.From, tx.To, tx.Amount))
}

// Block represents a single block in the blockchain.
// Each block contains a height (index), a Unix timestamp, a list of
// transactions, the hash of the previous block, a nonce used for
// proof-of-work mining, and its own computed hash.
type Block struct {
	Height       int           `json:"height"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	MerkleRoot   string        `json:"merkle_root"`
	PrevHash     string        `json:"prev_hash"`
	Nonce        int           `json:"nonce"`
	Hash         string        `json:"hash"`
}

// hashInput is a helper struct used to build a stable, deterministic
// serialisation of block fields for hashing. The Hash field of the block
// is deliberately excluded so that we can compute (and later verify) the
// hash from the remaining fields.
type hashInput struct {
	Height     int    `json:"height"`
	Timestamp  int64  `json:"timestamp"`
	PrevHash   string `json:"prev_hash"`
	MerkleRoot string `json:"merkle_root"`
	Nonce      int    `json:"nonce"`
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
		Height:     b.Height,
		Timestamp:  b.Timestamp,
		PrevHash:   b.PrevHash,
		MerkleRoot: b.MerkleRoot,
		Nonce:      b.Nonce,
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
		MerkleRoot:   ComputeMerkleRoot([]Transaction{}),
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
		MerkleRoot:   ComputeMerkleRoot(txns),
		PrevHash:     prevHash,
	}

	start := time.Now()
	var totalAttempts int64

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	type result struct {
		nonce int
		hash  string
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resCh := make(chan result)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			localBlock := b
			localBlock.Nonce = workerID
			var localAttempts int64

			for {
				// Check for cancellation every 1000 iterations to avoid select overhead on every loop
				if localAttempts%1000 == 0 {
					select {
					case <-ctx.Done():
						atomic.AddInt64(&totalAttempts, localAttempts)
						return
					default:
					}
				}

				h := ComputeHash(localBlock)
				localAttempts++

				if strings.HasPrefix(h, prefix) {
					atomic.AddInt64(&totalAttempts, localAttempts)
					select {
					case resCh <- result{nonce: localBlock.Nonce, hash: h}:
					case <-ctx.Done():
					}
					return
				}
				localBlock.Nonce += numWorkers
			}
		}(i)
	}

	res := <-resCh
	cancel() // Stop other workers

	b.Nonce = res.nonce
	b.Hash = res.hash

	return b, int(atomic.LoadInt64(&totalAttempts)), time.Since(start)
}

// ComputeMerkleRoot computes a simple Merkle root from a list of transactions.
func ComputeMerkleRoot(txns []Transaction) string {
	if len(txns) == 0 {
		return strings.Repeat("0", 64)
	}

	var hashes []string
	for _, tx := range txns {
		data, _ := json.Marshal(tx)
		h := sha256.Sum256(data)
		hashes = append(hashes, fmt.Sprintf("%x", h))
	}

	for len(hashes) > 1 {
		var nextLevel []string
		for i := 0; i < len(hashes); i += 2 {
			var combined []byte
			if i+1 < len(hashes) {
				combined = append([]byte(hashes[i]), []byte(hashes[i+1])...)
			} else {
				combined = append([]byte(hashes[i]), []byte(hashes[i])...) // duplicate last if odd
			}
			h := sha256.Sum256(combined)
			nextLevel = append(nextLevel, fmt.Sprintf("%x", h))
		}
		hashes = nextLevel
	}

	return hashes[0]
}
