// Package persist handles saving the blockchain to disk as JSON and
// reloading it on startup, so that state survives between program runs (FR-8).
package persist

import (
	"encoding/json"
	"fmt"
	"os"

	"blockchain/block"
)

// chainData is the on-disk representation. We store the blocks and any
// pending transactions so they survive between invocations.
type chainData struct {
	Blocks  []block.Block       `json:"blocks"`
	Pending []block.Transaction `json:"pending,omitempty"`
}

// Save writes the current chain of blocks and pending transactions to the
// given file path as pretty-printed JSON. It overwrites any existing file.
func Save(blocks []block.Block, pending []block.Transaction, filePath string) error {
	data := chainData{Blocks: blocks, Pending: pending}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("persist: marshal error: %w", err)
	}

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("persist: write error: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("persist: rename error: %w", err)
	}

	return nil
}

// Load reads a chain of blocks and pending transactions from the given JSON
// file. If the file does not exist, it returns nil with no error (the caller
// should initialise a fresh chain). Any other I/O or parsing error is returned.
func Load(filePath string) ([]block.Block, []block.Transaction, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil // no saved state — start fresh
		}
		return nil, nil, fmt.Errorf("persist: read error: %w", err)
	}

	var data chainData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, nil, fmt.Errorf("persist: unmarshal error: %w", err)
	}

	return data.Blocks, data.Pending, nil
}
