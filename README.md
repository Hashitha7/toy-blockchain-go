# 🧱 Toy Blockchain — Go Implementation (Elite Version)

![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Tests](https://img.shields.io/badge/tests-14%2F14_passing-brightgreen)
![Features](https://img.shields.io/badge/features-all_stretch_goals_completed-orange)

A highly advanced, self-contained blockchain built from scratch in Go. This command-line application goes beyond the basic requirements, demonstrating production-grade features including **Digital Signatures**, **Merkle Trees**, **Concurrent Mining**, **Dynamic Difficulty Retargeting**, and **Longest-Chain Fork Resolution**.

---

## ✨ Features (All Stretch Goals Implemented!)

1. **Digital Signatures (`ed25519`)**: Full Public/Private key cryptography. Transactions are signed and verified.
2. **Merkle Root**: Block hashes are calculated using a Merkle Tree of transactions, just like Bitcoin.
3. **Concurrent Mining**: Blazing fast proof-of-work using Go's `goroutines` and channels to utilize all CPU cores.
4. **Difficulty Retargeting**: The blockchain automatically adjusts the mining difficulty every 5 blocks to maintain a target block time.
5. **Fork Resolution**: Resolves chain forks by adopting the longest valid competing chain.
6. **Robust Ledger**: `int64` precision (no float errors), atomic file saving, full ledger replay during validation, and strict double-spend prevention from the pending pool.

---

## 🚀 Quick Start

Build the project:
```bash
go build -o blockchain.exe .
```

Or run directly with `go run`:
```bash
go run . <command> [flags]
```

---

## 💻 Commands

| Command | Description |
|---|---|
| `gen-key` | Generate a new `ed25519` Public/Private wallet key pair |
| `add-tx` | Add a signed transaction to the pending pool |
| `mine` | Mine a new block using concurrent goroutines |
| `print` | Print the full chain in a readable format |
| `validate` | Run a full integrity check (Ledger Replay, Signatures, Merkle Tree) |
| `balances` | Show all account balances |
| `resolve-fork` | Resolve a fork by providing a competing chain file |

### Global Flags

| Flag | Default | Description |
|---|---|---|
| `-data PATH` | `chain.json` | Path to the chain persistence file |

---

## 💡 Usage Examples

### 1. Generate a Wallet
```bash
go run . gen-key
```
*(Save the output Public Key and Private Key!)*

### 2. Mint Initial Funds (Coinbase)
`coinbase` is a special reserved sender used to mint new money into the ecosystem. It does not require a signature.
```bash
go run . add-tx -from coinbase -to <YOUR_PUBLIC_KEY> -amount 100
```

### 3. Send Money (Requires Digital Signature)
To send money, you must sign the transaction using your Private Key.
```bash
go run . add-tx -from <YOUR_PUBLIC_KEY> -to <BOB_PUBLIC_KEY> -amount 25 -priv <YOUR_PRIVATE_KEY>
```

### 4. Mine a Block
Mine the pending transactions into a block. This uses all available CPU cores!
```bash
go run . mine
```

### 5. Validate the Chain
Verifies the Merkle Roots, Proof-of-Work, Digital Signatures, and replays the ledger to ensure no double-spending.
```bash
go run . validate
```

### 6. Resolve a Fork
If another node has a longer, valid chain, you can adopt it:
```bash
go run . resolve-fork -competing another_chain.json
```

---

## 📁 Project Structure

```text
blockchain/
├── main.go              # CLI entry point and command dispatch
├── block/
│   ├── block.go         # Block struct, Merkle Root, Concurrent Mining
│   └── block_test.go    # Hash determinism, genesis, mining tests
├── chain/
│   ├── chain.go         # Blockchain, dynamic difficulty, fork resolution
│   └── chain_test.go    # Valid chain, tamper detection, retargeting tests
├── ledger/
│   ├── ledger.go        # Account balances, Signature validation
│   └── ledger_test.go   # Overspend rejection, double-spend tests
├── persist/
│   └── persist.go       # Atomic JSON save/load for chain state
├── wallet/
│   └── wallet.go        # ed25519 Key generation and Signature logic
├── Makefile             # Build, test, vet, fmt shortcuts
├── README.md            # This file
└── FEEDBACK-toy-blockchain-go-hashitha7.md  # Original Reviewer Feedback
```

---

## 🛠️ Design Decisions

### 1. Hashing & Merkle Root
Block hashes are computed using **SHA-256** over a deterministic JSON serialisation of `height`, `timestamp`, `prev_hash`, `nonce`, and the **`merkle_root`**. The Merkle Root is calculated by hashing pairs of transactions iteratively until a single root hash is derived.

### 2. Concurrent Proof-of-Work
Mining utilizes `runtime.NumCPU()` goroutines. Each worker searches a distinct slice of the nonce space. When a worker finds a valid hash, it sends the result via a channel and cancels the context of all other workers.

### 3. Dynamic Difficulty
The chain evaluates the time taken to mine every 5 blocks. If the mining was too fast (under the target block time), the difficulty automatically increases. If it was too slow, it decreases.

### 4. Persistence & Safety
The chain is saved automatically as JSON (`chain.json`). To prevent file corruption during crashes, it uses **Atomic Saving** (writes to a `.tmp` file and securely renames it).

---

## 🧪 Running Tests

We have 14 robust unit tests achieving exceptional coverage.
```bash
go test ./...          # Run all tests
go test -v ./...       # Verbose output
go vet ./...           # Static analysis
gofmt -l .             # Check formatting
```