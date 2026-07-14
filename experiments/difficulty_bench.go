// experiments/difficulty_bench.go — runs the difficulty vs effort experiment
// and prints results as a Markdown table for the research report.
package main

import (
	"fmt"
	"strings"
	"time"

	"blockchain/block"
)

func main() {
	fmt.Println("## Difficulty vs Effort Experiment")
	fmt.Println()
	fmt.Println("| Difficulty | Leading Zeros | Hashes Tried | Time Taken | Expected (16^d) |")
	fmt.Println("|:----------:|:-------------:|:------------:|:----------:|:---------------:|")

	txns := []block.Transaction{
		{From: "coinbase", To: "Alice", Amount: 100},
	}
	prevHash := strings.Repeat("0", 64)

	for diff := 1; diff <= 5; diff++ {
		// Run 3 trials and take median
		type trial struct {
			attempts int
			elapsed  time.Duration
		}
		trials := make([]trial, 3)
		for i := 0; i < 3; i++ {
			_, attempts, elapsed := block.MineBlock(1, prevHash, txns, diff)
			trials[i] = trial{attempts, elapsed}
		}
		// Sort by attempts to find median
		for i := 0; i < 2; i++ {
			for j := i + 1; j < 3; j++ {
				if trials[j].attempts < trials[i].attempts {
					trials[i], trials[j] = trials[j], trials[i]
				}
			}
		}
		median := trials[1]

		expected := 1
		for i := 0; i < diff; i++ {
			expected *= 16
		}

		fmt.Printf("| %d          | `%s` | %d | %s | %d |\n",
			diff,
			strings.Repeat("0", diff),
			median.attempts,
			median.elapsed.Round(time.Microsecond),
			expected,
		)
	}

	fmt.Println()
	fmt.Println("## Tamper Evidence Experiment")
	fmt.Println()

	// Build a chain of 3 blocks
	fmt.Println("### Before tampering:")
	fmt.Println("```")

	genesis := block.GenesisBlock()
	b1, _, _ := block.MineBlock(1, genesis.Hash, []block.Transaction{
		{From: "coinbase", To: "Alice", Amount: 100},
	}, 2)
	b2, _, _ := block.MineBlock(2, b1.Hash, []block.Transaction{
		{From: "Alice", To: "Bob", Amount: 25},
	}, 2)

	blocks := []block.Block{genesis, b1, b2}

	// Validate before tamper
	fmt.Printf("Chain has %d blocks\n", len(blocks))
	fmt.Printf("Block 0 hash: %s\n", blocks[0].Hash[:16]+"...")
	fmt.Printf("Block 1 hash: %s\n", blocks[1].Hash[:16]+"...")
	fmt.Printf("Block 1 txn:  coinbase -> Alice : 100.00\n")
	fmt.Printf("Block 2 hash: %s\n", blocks[2].Hash[:16]+"...")
	valid := validateBlocks(blocks, 2)
	fmt.Printf("Validation: %s\n", valid)
	fmt.Println("```")

	// Tamper with block 1
	fmt.Println()
	fmt.Println("### After tampering (change Block 1 transaction amount 100 → 999999):")
	fmt.Println("```")
	blocks[1].Transactions[0].Amount = 999999
	fmt.Printf("Block 1 txn:  coinbase -> Alice : 999999.00 (TAMPERED)\n")
	recomputed := block.ComputeHash(blocks[1])
	fmt.Printf("Block 1 stored hash:   %s\n", blocks[1].Hash[:16]+"...")
	fmt.Printf("Block 1 computed hash: %s\n", recomputed[:16]+"...")
	fmt.Printf("Match: %v\n", blocks[1].Hash == recomputed)
	valid = validateBlocks(blocks, 2)
	fmt.Printf("Validation: %s\n", valid)
	fmt.Println("```")
}

func validateBlocks(blocks []block.Block, difficulty int) string {
	prefix := strings.Repeat("0", difficulty)
	for i, b := range blocks {
		recomputed := block.ComputeHash(b)
		if b.Hash != recomputed {
			return fmt.Sprintf("INVALID at block %d — hash mismatch (stored: %s..., computed: %s...)",
				i, b.Hash[:16], recomputed[:16])
		}
		if i > 0 {
			if b.PrevHash != blocks[i-1].Hash {
				return fmt.Sprintf("INVALID at block %d — prev_hash does not match block %d hash", i, i-1)
			}
			if !strings.HasPrefix(b.Hash, prefix) {
				return fmt.Sprintf("INVALID at block %d — hash does not meet difficulty target", i)
			}
		}
	}
	return "VALID ✓"
}
