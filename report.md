# Research Report — Toy Blockchain

**Author:** Hashitha  
**Date:** July 2026  
**Module:** blockchain (Go 1.26)

---

## 1. Introduction

This report accompanies a minimal blockchain implementation built from scratch
in Go. The system maintains an append-only chain of blocks, each containing
transactions, linked by SHA-256 hashes, and secured by a proof-of-work
mechanism. This report presents three required investigations and answers the
discussion questions posed in the assessment brief.

---

## 2. Experiment 1 — Tamper Evidence

### Method

1. Built a chain of 3 blocks:
   - Block 0: Genesis (no transactions)
   - Block 1: coinbase → Alice: 100
   - Block 2: Alice → Bob: 25

2. Ran `validate` to confirm the honest chain passes.

3. Opened `chain.json` and manually changed Block 1's transaction amount
   from `100` to `999999`.

4. Ran `validate` again.

### Results

**Before tampering:**
```
Chain has 3 blocks
Block 0 hash: 258a64c0592be20a...
Block 1 hash: 00d7edc9f142bd6c...
Block 1 txn:  coinbase -> Alice : 100.00
Block 2 hash: 00a018e95cda47dc...
Validation: VALID ✓
```

**After tampering (changed Block 1 transaction amount from 100 to 999999):**
```
Block 1 txn:  coinbase -> Alice : 999999.00 (TAMPERED)
Block 1 stored hash:   00d7edc9f142bd6c...
Block 1 computed hash: eeb0d8f9ecfbd735...
Match: false
Validation: INVALID at block 1 — hash mismatch
  (stored: 00d7edc9f142bd6c..., computed: eeb0d8f9ecfbd735...)
```

### Analysis

The validation routine recomputes the SHA-256 hash of each block from its
fields (height, timestamp, prev_hash, nonce, transactions). When the
transaction amount was changed from 100 to 999999, the recomputed hash no
longer matched the stored hash. The chain was rejected at **Block 1** — the
exact block that was tampered with.

Note that even if the attacker also updated Block 1's stored hash to match the
new content, Block 2's `prev_hash` field would then fail to match, causing
validation to fail at Block 2. This cascading dependency is the fundamental
integrity guarantee of a hash-linked chain.

---

## 3. Experiment 2 — Difficulty vs. Effort

### Method

Mined a single block (with one coinbase transaction) at difficulty levels 1
through 5, recording the number of SHA-256 hashes computed and the wall-clock
time taken. Each measurement was repeated 3 times and the median reported.

### Results

| Difficulty | Leading Zeros | Median Hashes | Median Time   | Expected (16^d) |
|:----------:|:-------------:|:-------------:|:--------------|:---------------:|
| 1          | `0`           | 10            | < 1 ms        | 16              |
| 2          | `00`          | 202           | < 1 ms        | 256             |
| 3          | `000`         | 7,408         | 7.765 ms      | 4,096           |
| 4          | `0000`        | 122,241       | 105.421 ms    | 65,536          |
| 5          | `00000`       | 1,321,602     | 1.206 s       | 1,048,576       |

### Analysis

The number of hash attempts grows **exponentially** with difficulty, not
linearly. Each additional leading zero digit requires (on average) 16× more
attempts, because each hex digit has a 1/16 probability of being zero. The
expected number of hashes is approximately **16^d** where d is the difficulty.

This exponential growth is why Bitcoin's difficulty adjustment is so powerful:
small changes in difficulty produce large changes in required computational
work, making it easy to calibrate block times across a wide range of hash
rates.

The variance is also significant — some runs find a valid nonce much faster or
slower than the expected value. This is characteristic of the random search
process (each hash attempt is essentially an independent Bernoulli trial).

---

## 4. Design Write-Up

### Hashing Scheme

Each block's hash is computed as:

```
SHA-256( JSON( {height, timestamp, prev_hash, nonce, transactions} ) )
```

**Field order:** height → timestamp → prev_hash → nonce → transactions

The block's own `hash` field is excluded from the input to avoid circularity.
Go's `encoding/json.Marshal` serialises struct fields in their declaration
order, ensuring determinism. The `hashInput` struct is a separate type (not the
`Block` type) to explicitly control which fields are hashed and in what order.

### Why This Guarantees Integrity

1. **Intra-block integrity:** If any field in a block is changed, the
   recomputed hash will differ from the stored hash. Validation catches this.

2. **Inter-block integrity:** Each block stores the hash of the previous block
   in its `prev_hash` field. If Block N is modified, its hash changes, which
   means Block N+1's `prev_hash` no longer matches. This creates a **cascade
   of failures** — modifying any block invalidates all subsequent blocks.

3. **Proof-of-work binding:** The nonce is included in the hash input. Finding
   a nonce that produces a hash with the required leading zeros is
   computationally expensive, so an attacker would need to re-mine every block
   from the tampered point onward.

---

## 5. Discussion Questions

### 5.1 Why does the previous-hash link make tampering impractical in a real chain?

In our local toy, we can modify a block and simply re-mine all subsequent
blocks because there is only one node and difficulty is low. In a real
blockchain (e.g., Bitcoin):

- **Distributed consensus:** Thousands of nodes independently validate the
  chain. An attacker must convince >50% of the network to accept a forged
  chain (51% attack).
- **Cumulative proof-of-work:** Re-mining from a tampered block means redoing
  all the work from that point to the chain tip. At Bitcoin's difficulty, this
  requires more computational power than the rest of the network combined.
- **Network propagation:** Honest miners continue extending the real chain. The
  attacker must outpace the entire network, which is economically infeasible
  for all but the most well-funded adversaries.

### 5.2 Proof-of-Work vs. Alternatives

**Proof-of-Stake (PoS)** is a major alternative where validators are selected
based on the amount of cryptocurrency they "stake" (lock up) as collateral.

| Aspect       | Proof-of-Work         | Proof-of-Stake          |
|--------------|-----------------------|-------------------------|
| **Advantage**| Battle-tested security; no stake needed to participate | Energy-efficient; ~99.9% less electricity than PoW |
| **Drawback** | Enormous energy consumption (Bitcoin uses ~150 TWh/yr) | "Nothing at stake" problem — validators can cheaply vote on multiple chain forks without penalty in naive implementations |

### 5.3 Three Ways This Toy Differs from a Production Blockchain

1. **No peer-to-peer consensus:** A production blockchain distributes the chain
   across thousands of nodes that gossip blocks and independently validate.
   Our toy is single-process with no network.

2. **No transaction signatures:** Real blockchains require transactions to be
   signed with the sender's private key (e.g., ECDSA in Bitcoin). Our toy
   trusts the `from` field, which means anyone can spend anyone's balance.

3. **No Merkle tree:** Production blockchains summarise transactions into a
   Merkle root hash, enabling efficient proofs that a specific transaction is
   included in a block without downloading all transactions (SPV proofs). Our
   toy hashes the full transaction list as a flat JSON array.

**Sketch: Adding Transaction Signatures (Item 2)**

To add digital signatures, I would:

1. Generate an ECDSA key pair (private + public key) for each account using
   Go's `crypto/ecdsa` and `crypto/elliptic` packages.
2. Account addresses become the hash of the public key.
3. When creating a transaction, the sender signs `SHA-256(from + to + amount)`
   with their private key.
4. The `Transaction` struct gains a `Signature` field (hex-encoded).
5. During validation, each transaction's signature is verified against the
   sender's public key before accepting it into a block.

This would prevent unauthorised spending without the sender's private key.

---

## 6. References

- Nakamoto, S. (2008). *Bitcoin: A Peer-to-Peer Electronic Cash System*.
  https://bitcoin.org/bitcoin.pdf
- Go standard library documentation: `crypto/sha256`, `encoding/json`.
  https://pkg.go.dev/std
- Ethereum Foundation. *Proof-of-Stake FAQ*.
  https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/
