// Package ledger tracks account balances derived from the blockchain and
// validates transactions before they are included in a block.
package ledger

import (
	"fmt"

	"blockchain/block"
)

// Ledger maintains the balance of every account that has ever sent or
// received funds. It is rebuilt by replaying all transactions in the chain.
type Ledger struct {
	Balances map[string]int64
}

// NewLedger creates a new, empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		Balances: make(map[string]int64),
	}
}

// ValidateTransaction checks whether a transaction is well-formed and whether
// the sender has sufficient funds. It returns an error describing the reason
// for rejection, or nil if the transaction is acceptable.
//
// Rules:
//   - Amount must be positive (> 0).
//   - The special sender "coinbase" can mint unlimited funds (faucet).
//   - Any other sender must have a balance >= Amount.
func (l *Ledger) ValidateTransaction(tx block.Transaction, pending []block.Transaction) error {
	if tx.Amount <= 0 {
		return fmt.Errorf("invalid amount: %d (must be positive)", tx.Amount)
	}

	if tx.From == "" || tx.To == "" {
		return fmt.Errorf("sender and recipient must not be empty")
	}

	if tx.To == "coinbase" {
		return fmt.Errorf("coinbase is a reserved sender, cannot be used as a recipient")
	}

	// Coinbase transactions can create money out of thin air.
	if tx.From == "coinbase" {
		return nil
	}

	// Calculate available balance: on-chain balance minus already pending spends.
	available := l.Balances[tx.From]
	for _, ptx := range pending {
		if ptx.From == tx.From {
			available -= ptx.Amount
		}
	}

	if available < tx.Amount {
		return fmt.Errorf("insufficient balance: %s has %d available (on-chain minus pending), tried to send %d", tx.From, available, tx.Amount)
	}

	return nil
}

// ApplyTransaction updates the ledger balances for a validated transaction.
// The caller must validate the transaction first; this method does not
// re-check balances.
func (l *Ledger) ApplyTransaction(tx block.Transaction) {
	if tx.From != "coinbase" {
		l.Balances[tx.From] -= tx.Amount
	}
	l.Balances[tx.To] += tx.Amount
}

// RebuildFromBlocks replays every transaction in the given slice of blocks
// (in order) to reconstruct the full ledger state. This is called at startup
// when loading a persisted chain.
func (l *Ledger) RebuildFromBlocks(blocks []block.Block) {
	l.Balances = make(map[string]int64)
	for _, b := range blocks {
		for _, tx := range b.Transactions {
			l.ApplyTransaction(tx)
		}
	}
}
