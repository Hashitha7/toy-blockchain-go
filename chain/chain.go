// Package chain manages the ordered sequence of blocks that form the
// blockchain. It handles block creation, mining, full-chain validation,
// and tamper detection.
package chain

import (
	"fmt"
	"strings"
	"time"

	"blockchain/block"
	"blockchain/ledger"
)

const (
	BaseDifficulty               = 2 // Base difficulty
	TargetBlockTime              = 2 // Target block time in seconds
	DifficultyAdjustmentInterval = 5 // Adjust difficulty every 5 blocks
)

// Chain holds the full sequence of mined blocks and a pool of pending
// transactions that have not yet been included in a block.
type Chain struct {
	Blocks        []block.Block       `json:"blocks"`
	Pending       []block.Transaction `json:"-"` // not persisted; drained on mine
	MaxTxPerBlock int                 `json:"-"` // max transactions per block (0 = unlimited)
}

// NewChain creates a fresh blockchain initialised with the deterministic
// genesis block (FR-2). All other blocks descend from it.
func NewChain() *Chain {
	return &Chain{
		Blocks:  []block.Block{block.GenesisBlock()},
		Pending: []block.Transaction{},
	}
}

// GetDifficulty calculates the required difficulty for a given height,
// adjusting based on the time taken to mine the previous blocks.
func (c *Chain) GetDifficulty(height int) int {
	if height < DifficultyAdjustmentInterval {
		return BaseDifficulty
	}

	diff := BaseDifficulty
	boundary := (height / DifficultyAdjustmentInterval) * DifficultyAdjustmentInterval

	for h := DifficultyAdjustmentInterval; h <= boundary; h += DifficultyAdjustmentInterval {
		if h-1 >= len(c.Blocks) || h-DifficultyAdjustmentInterval >= len(c.Blocks) {
			break
		}
		lastBlock := c.Blocks[h-1]
		firstBlock := c.Blocks[h-DifficultyAdjustmentInterval]

		timeExpected := int64(DifficultyAdjustmentInterval * TargetBlockTime)
		timeTaken := lastBlock.Timestamp - firstBlock.Timestamp

		if timeTaken < timeExpected/2 {
			diff++
		} else if timeTaken > timeExpected*2 {
			if diff > 1 {
				diff--
			}
		}
	}
	return diff
}

// AddTransaction validates a transaction against the current ledger state
// and, if accepted, adds it to the pending pool. Returns an error if the
// transaction is rejected.
func (c *Chain) AddTransaction(tx block.Transaction, l *ledger.Ledger) error {
	if err := l.ValidateTransaction(tx, c.Pending); err != nil {
		return err
	}
	c.Pending = append(c.Pending, tx)
	return nil
}

// MineNextBlock takes transactions from the pending pool, mines a new block
// with the configured difficulty, applies the transactions to the ledger,
// and appends the block to the chain.
//
// It returns the mined block, the number of hash attempts, and the mining
// duration. An error is returned if there are no pending transactions.
func (c *Chain) MineNextBlock(l *ledger.Ledger) (block.Block, int, time.Duration, error) {
	if len(c.Pending) == 0 {
		return block.Block{}, 0, 0, fmt.Errorf("no pending transactions to mine")
	}

	// Determine which transactions go into this block.
	txns := c.Pending
	if c.MaxTxPerBlock > 0 && len(txns) > c.MaxTxPerBlock {
		txns = txns[:c.MaxTxPerBlock]
	}

	lastBlock := c.Blocks[len(c.Blocks)-1]
	newHeight := lastBlock.Height + 1
	difficulty := c.GetDifficulty(newHeight)

	minedBlock, attempts, elapsed := block.MineBlock(newHeight, lastBlock.Hash, txns, difficulty)

	// Apply transactions to the ledger.
	for _, tx := range txns {
		l.ApplyTransaction(tx)
	}

	c.Blocks = append(c.Blocks, minedBlock)

	// Remove mined transactions from the pending pool.
	c.Pending = c.Pending[len(txns):]

	return minedBlock, attempts, elapsed, nil
}

// ValidationResult holds the outcome of a full-chain validation check.
type ValidationResult struct {
	Valid        bool
	ErrorBlock   int // height of first invalid block (-1 if valid)
	ErrorMessage string
}

// Validate performs a full integrity check of the entire chain (FR-6).
// It verifies that:
//   - Each block's stored hash matches its recomputed hash.
//   - Each block's PrevHash matches the previous block's stored hash.
//   - Each block's hash satisfies the proof-of-work difficulty target.
//   - Heights are sequential (0, 1, 2, …).
//   - Timestamps are non-decreasing.
//
// On the first failure, it returns a result identifying the offending block.
func (c *Chain) Validate() ValidationResult {
	if len(c.Blocks) == 0 {
		return ValidationResult{Valid: false, ErrorBlock: -1, ErrorMessage: "chain is empty"}
	}

	testLedger := ledger.NewLedger()

	for i, b := range c.Blocks {
		// Check height is sequential.
		if b.Height != i {
			return ValidationResult{
				Valid:        false,
				ErrorBlock:   i,
				ErrorMessage: fmt.Sprintf("block %d has height %d, expected %d", i, b.Height, i),
			}
		}

		// Verify Merkle Root matches the transactions
		if b.MerkleRoot != block.ComputeMerkleRoot(b.Transactions) {
			return ValidationResult{
				Valid:        false,
				ErrorBlock:   i,
				ErrorMessage: "merkle root does not match transactions",
			}
		}

		// Check stored hash matches recomputed hash.
		recomputed := block.ComputeHash(b)
		if b.Hash != recomputed {
			return ValidationResult{
				Valid:        false,
				ErrorBlock:   i,
				ErrorMessage: fmt.Sprintf("block %d hash mismatch: stored %s, computed %s", i, b.Hash, recomputed),
			}
		}

		// Check previous-hash link (genesis has the well-known zeroed hash).
		if i == 0 {
			if b.PrevHash != strings.Repeat("0", 64) {
				return ValidationResult{
					Valid:        false,
					ErrorBlock:   0,
					ErrorMessage: "genesis block has unexpected prev_hash",
				}
			}
		} else {
			if b.PrevHash != c.Blocks[i-1].Hash {
				return ValidationResult{
					Valid:        false,
					ErrorBlock:   i,
					ErrorMessage: fmt.Sprintf("block %d prev_hash does not match block %d hash", i, i-1),
				}
			}
		}

		// Check proof-of-work target (skip genesis which may not need mining).
		if i > 0 {
			diff := c.GetDifficulty(i)
			prefix := strings.Repeat("0", diff)
			if !strings.HasPrefix(b.Hash, prefix) {
				return ValidationResult{
					Valid:        false,
					ErrorBlock:   i,
					ErrorMessage: fmt.Sprintf("block %d hash does not meet difficulty target (need prefix %q)", i, prefix),
				}
			}
		}

		// Check timestamps are non-decreasing.
		if i > 0 && b.Timestamp < c.Blocks[i-1].Timestamp {
			return ValidationResult{
				Valid:        false,
				ErrorBlock:   i,
				ErrorMessage: fmt.Sprintf("block %d timestamp (%d) is before block %d timestamp (%d)", i, b.Timestamp, i-1, c.Blocks[i-1].Timestamp),
			}
		}

		// Replay transactions to check ledger validity (no overspends, no negative amounts)
		for _, tx := range b.Transactions {
			// Pass nil for pending since we are replaying confirmed blocks
			if err := testLedger.ValidateTransaction(tx, nil); err != nil {
				return ValidationResult{
					Valid:        false,
					ErrorBlock:   i,
					ErrorMessage: fmt.Sprintf("invalid transaction: %v", err),
				}
			}
			testLedger.ApplyTransaction(tx)
		}
	}

	return ValidationResult{Valid: true, ErrorBlock: -1}
}

// PrintChain returns a human-readable representation of the entire chain.
func (c *Chain) PrintChain() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Blockchain (%d blocks)\n", len(c.Blocks)))
	sb.WriteString(strings.Repeat("=", 70) + "\n")

	for _, b := range c.Blocks {
		sb.WriteString(fmt.Sprintf("\n--- Block %d ---\n", b.Height))
		sb.WriteString(fmt.Sprintf("  Timestamp : %d (%s)\n", b.Timestamp, time.Unix(b.Timestamp, 0).UTC().Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("  PrevHash  : %s\n", b.PrevHash))
		sb.WriteString(fmt.Sprintf("  Hash      : %s\n", b.Hash))
		sb.WriteString(fmt.Sprintf("  Nonce     : %d\n", b.Nonce))
		sb.WriteString(fmt.Sprintf("  Txns      : %d\n", len(b.Transactions)))
		for j, tx := range b.Transactions {
			sb.WriteString(fmt.Sprintf("    [%d] %s -> %s : %d\n", j, tx.From, tx.To, tx.Amount))
		}
	}

	return sb.String()
}

// ResolveFork implements the longest-valid-chain rule.
// If the competingBlocks represent a valid chain that is longer than the current chain,
// this node adopts the competing chain and rebuilds its ledger.
func (c *Chain) ResolveFork(competingBlocks []block.Block, l *ledger.Ledger) bool {
	if len(competingBlocks) <= len(c.Blocks) {
		return false // must be strictly longer
	}

	// Create a temporary chain to validate the competing blocks.
	tempChain := &Chain{
		Blocks:  competingBlocks,
		Pending: nil,
	}

	res := tempChain.Validate()
	if !res.Valid {
		return false // reject invalid competing chain
	}

	// It's valid and longer! Adopt it.
	c.Blocks = competingBlocks

	// Rebuild the ledger.
	l.RebuildFromBlocks(c.Blocks)

	// Clean up pending transactions
	c.Pending = []block.Transaction{}

	return true
}
