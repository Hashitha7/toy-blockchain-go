// Toy Blockchain — a minimal command-line blockchain and ledger simulator.
//
// Usage:
//
//	blockchain [flags] <command> [command-flags]
//
// Commands:
//
//	add-tx    Add a transaction to the pending pool
//	mine      Mine a new block from pending transactions
//	print     Print the full chain in human-readable form
//	validate  Validate the integrity of the entire chain
//	balances  Show all account balances
//
// Global flags:
//
//	-difficulty N   Proof-of-work difficulty (leading hex zeros, default 3)
//	-data PATH      Path to the chain data file (default chain.json)
//	-maxtx N        Max transactions per block (default 0 = unlimited)
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"blockchain/block"
	"blockchain/chain"
	"blockchain/ledger"
	"blockchain/persist"
)

func main() {
	// ── Global flags ──────────────────────────────────────────────────────
	difficulty := flag.Int("difficulty", 3, "Proof-of-work difficulty (number of leading hex zeros)")
	dataFile := flag.String("data", "chain.json", "Path to the chain data file")
	maxTx := flag.Int("maxtx", 0, "Maximum transactions per block (0 = unlimited)")

	// We need to parse global flags before extracting the sub-command.
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	// ── Load or initialise chain ──────────────────────────────────────────
	bc := chain.NewChain(*difficulty)
	bc.MaxTxPerBlock = *maxTx

	savedBlocks, savedPending, err := persist.Load(*dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading chain: %v\n", err)
		os.Exit(1)
	}
	if savedBlocks != nil {
		bc.Blocks = savedBlocks
	}
	if savedPending != nil {
		bc.Pending = savedPending
	}

	// Rebuild ledger from persisted blocks.
	l := ledger.NewLedger()
	l.RebuildFromBlocks(bc.Blocks)

	// Pre-apply coinbase transactions from the pending pool so that
	// funds minted in a previous add-tx invocation are available for
	// spending before mining.
	for _, tx := range bc.Pending {
		if tx.From == "coinbase" {
			l.ApplyTransaction(tx)
		}
	}

	// ── Dispatch command ──────────────────────────────────────────────────
	switch command {
	case "add-tx":
		cmdAddTx(args[1:], bc, l, *dataFile)
	case "mine":
		cmdMine(bc, l, *dataFile)
	case "print":
		cmdPrint(bc)
	case "validate":
		cmdValidate(bc)
	case "balances":
		cmdBalances(l)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

// ── Sub-command implementations ──────────────────────────────────────────

func cmdAddTx(args []string, bc *chain.Chain, l *ledger.Ledger, dataFile string) {
	fs := flag.NewFlagSet("add-tx", flag.ExitOnError)
	from := fs.String("from", "", "Sender address (use 'coinbase' to mint)")
	to := fs.String("to", "", "Recipient address")
	amount := fs.Float64("amount", 0, "Amount to transfer")
	fs.Parse(args)

	if *from == "" || *to == "" || *amount == 0 {
		fmt.Fprintln(os.Stderr, "Usage: blockchain add-tx -from SENDER -to RECIPIENT -amount VALUE")
		os.Exit(1)
	}

	tx := block.Transaction{From: *from, To: *to, Amount: *amount}

	if err := bc.AddTransaction(tx, l); err != nil {
		fmt.Fprintf(os.Stderr, "Transaction rejected: %v\n", err)
		os.Exit(1)
	}

	// For coinbase, apply immediately so subsequent add-tx in the same
	// session can spend the minted funds before mining.
	if tx.From == "coinbase" {
		l.ApplyTransaction(tx)
	}

	fmt.Printf("✓ Transaction added to pending pool: %s → %s : %.2f\n", tx.From, tx.To, tx.Amount)
	fmt.Printf("  Pending transactions: %d\n", len(bc.Pending))

	// Persist pending transactions so they survive between invocations.
	if err := persist.Save(bc.Blocks, bc.Pending, dataFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}
}

func cmdMine(bc *chain.Chain, l *ledger.Ledger, dataFile string) {
	if len(bc.Pending) == 0 {
		fmt.Fprintln(os.Stderr, "No pending transactions. Use 'add-tx' first.")
		os.Exit(1)
	}

	fmt.Printf("Mining block %d (difficulty %d)...\n", len(bc.Blocks), bc.Difficulty)

	minedBlock, attempts, elapsed, err := bc.MineNextBlock(l)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mining failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Block %d mined!\n", minedBlock.Height)
	fmt.Printf("  Hash     : %s\n", minedBlock.Hash)
	fmt.Printf("  Nonce    : %d\n", minedBlock.Nonce)
	fmt.Printf("  Attempts : %d\n", attempts)
	fmt.Printf("  Time     : %s\n", elapsed.Round(1_000_000))
	fmt.Printf("  Txns     : %d\n", len(minedBlock.Transactions))

	// Save to disk.
	if err := persist.Save(bc.Blocks, bc.Pending, dataFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save chain: %v\n", err)
	} else {
		fmt.Printf("  Saved to : %s\n", dataFile)
	}
}

func cmdPrint(bc *chain.Chain) {
	fmt.Print(bc.PrintChain())
}

func cmdValidate(bc *chain.Chain) {
	result := bc.Validate()
	if result.Valid {
		fmt.Println("✓ CHAIN VALID")
		fmt.Printf("  Blocks: %d\n", len(bc.Blocks))
	} else {
		fmt.Println("✗ CHAIN INVALID")
		fmt.Printf("  First invalid block: %d\n", result.ErrorBlock)
		fmt.Printf("  Reason: %s\n", result.ErrorMessage)
		os.Exit(1)
	}
}

func cmdBalances(l *ledger.Ledger) {
	if len(l.Balances) == 0 {
		fmt.Println("No accounts found. Add some transactions first.")
		return
	}

	fmt.Println("Account Balances")
	fmt.Println(strings.Repeat("-", 40))
	for account, balance := range l.Balances {
		if account == "coinbase" {
			continue // don't show the coinbase pseudo-account
		}
		fmt.Printf("  %-20s : %.2f\n", account, balance)
	}
}

func printUsage() {
	fmt.Println("Toy Blockchain — a minimal blockchain and ledger simulator")
	fmt.Println()
	fmt.Println("Usage: blockchain [flags] <command> [command-flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  add-tx     Add a transaction to the pending pool")
	fmt.Println("  mine       Mine a new block from pending transactions")
	fmt.Println("  print      Print the full chain")
	fmt.Println("  validate   Validate chain integrity")
	fmt.Println("  balances   Show all account balances")
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  -difficulty N   PoW difficulty (default 3)")
	fmt.Println("  -data PATH     Data file path (default chain.json)")
	fmt.Println("  -maxtx N       Max txns per block (default 0 = unlimited)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blockchain add-tx -from coinbase -to Alice -amount 100")
	fmt.Println("  blockchain add-tx -from Alice -to Bob -amount 25")
	fmt.Println("  blockchain mine")
	fmt.Println("  blockchain print")
	fmt.Println("  blockchain validate")
	fmt.Println("  blockchain balances")
}
