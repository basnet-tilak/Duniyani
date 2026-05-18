 # About Duniyani

Duniyani is a fully modular Layer 1 blockchain blueprint written in modern Go (1.22+). It is specifically designed to serve as a high-performance foundation for Decentralized Physical Infrastructure Networks (DePIN). This document provides an in-depth look at the core logic, architectural decisions, and implementation details across the project.

## In-Depth Core Architecture & Implementation Logic

The codebase is strictly separated by domains to ensure a highly maintainable and pluggable infrastructure:

### 1. Blockchain & UTXO Model (`core`)
Duniyani employs an Unspent Transaction Output (UTXO) model. In this model, there are no "accounts" with balances; instead, balances are calculated by aggregating unspent outputs locked to a user's public key hash.
- **Transactions (`Transaction`, `TxInput`, `TxOutput`)**: 
  - `TxInput` points to a previous transaction's ID (`TxID`) and output index (`Vout`). It includes the spender's `PubKey` and a cryptographic `Signature`.
  - `TxOutput` contains a `Value` (denominated in 'Drops', where 1 DNY = 100,000,000 Drops) and a `PubKeyHash` locking the funds to the receiver.
- **Blocks (`Block`, `BlockHeader`)**: Blocks are cryptographically chained. The `BlockHeader` stores the `PrevBlockHash`, `Timestamp`, `Nonce`, and a `MerkleRoot`. 
- **Merkle Trees (`MerkleTree`, `MerkleNode`)**: To ensure transaction integrity, transactions are hashed into a binary Merkle tree bottom-up. Leaf nodes hash individual transaction byte payloads. Internal nodes recursively hash the concatenation of their left and right children (`sha256.Sum256(append(left, right))`).
- **UTXO Set (`UTXOSet`)**: To avoid scanning the entire blockchain history to calculate balances, the active, spendable outputs are cached in a dedicated database bucket (`ChainStateBucket`). When a block is added, the `UTXOSet` deletes the spent inputs and inserts the new unspent outputs.

### 2. Consensus Mechanism (`consensus`)
The blockchain uses a pluggable **Proof of Useful Work (PoUW)** engine.
- **Mining Process**: To mine a block, nodes must find a `Nonce` that, when combined with the block header and a specific `ComputeReceipt`, produces a hash whose numerical value is strictly less than the network's `DifficultyTarget`. 
- **Implementation**: The engine iteratively increments the `Nonce` in the `BlockHeader`, serializes the header, and applies a double SHA-256 hash. This mathematical puzzle secures the network against Sybil attacks and history rewriting.

### 3. Cryptography & Wallets (`crypto` & `wallet`)
The network leverages 2026-era standard cryptographic primitives:
- **ECDSA on secp256k1**: Wallets generate public and private key pairs using the standard Bitcoin elliptic curve. The implementation avoids deprecated standard library fields by utilizing `ecdh` and `x509.ParsePKIXPublicKey` with dynamic SubjectPublicKeyInfo (SPKI) prefixes for safe parsing and serialization.
- **Address Generation**: 
  1. The uncompressed public key is hashed using double SHA-256.
  2. The hash is truncated to 20 bytes to match standard legacy payload sizes.
  3. A version byte (`0x1e`) is prepended, pushing the final Base58 string to start with the character `"D"`.
  4. A 4-byte double-SHA256 checksum is appended to prevent typos.
  5. The entire payload is encoded into a Base58 string via big integer division math.
- **Transaction Signing**: The wallet signs a trimmed copy of the transaction (where input scripts are temporarily emptied) using the ECDSA private key.

### 4. Economics & Incentives (`economics`)
The economics engine strictly controls the issuance of new coins and fees:
- **Coinbase Transactions**: The first transaction in every block is a special `CoinbaseTx`. It has a single input with an empty `TxID` and a `Vout` of `-1`. This transaction mints the block reward and pays it to the miner's address.
- **Supply Halving Math**: The network starts with an `InitialBlockReward` of 50 DNY. The `BlockReward(height)` function calculates the halvings (`height / 210_000`). If halvings exceed 64, the reward drops to 0, capping the total supply.
- **Dynamic Fees**: `CalculateTransactionFee` uses `encoding/gob` to measure the exact byte size of a transaction. The required fee is `(size * FeePerByte) + (pendingMempoolTxCount * CongestionFeePerTx)`.

### 5. Mempool & State (`state` / `network`)
The `Mempool` acts as a thread-safe waiting room for transactions before they are confirmed in a block.
- **Validation & Double-Spend Protection**: 
  - When a transaction is submitted, the node queries the `database` for each input's `TxID:Vout` combination. 
  - It maintains an internal `spentInMempool` map to prevent two unconfirmed transactions from spending the same UTXO simultaneously.
  - It verifies that `Sum(Inputs) - Sum(Outputs) >= Required Dynamic Fee`.
- **Eviction**: Once a block is successfully mined and appended to the chain, the `main.go` loop iterates over the included transactions and explicitly removes them from the `Mempool` map.

### 6. P2P Networking (`network`)
Duniyani implements an abstracted Gossip/PubSub networking layer:
- **Wire Protocol**: Nodes exchange `Message` structs containing a byte-slice `Command` (`CmdVersion`, `CmdTx`, `CmdBlock`) and a `gob`-encoded payload.
- **Concurrency**: The `NetworkNode` runs in a dedicated goroutine, listening on an `incoming` channel. It utilizes `sync.RWMutex` to safely manage the `peers` map.
- **GossipSub Integration**: The blueprint includes a `MockP2PHost` implementing a topic-based publish/subscribe interface, laying the groundwork for integration with advanced networking stacks like `libp2p`.

### 7. Database Persistence (`database`)
State is saved locally using a concurrent-safe, file-backed Key-Value store. 
- **Storage Engine**: The `Database` struct utilizes an in-memory `map[string][]byte` protected by a `sync.RWMutex`.
- **File Serialization**: On startup, the map is hydrated from `duniyani_store.gob` using `gob.NewDecoder`. On every write (`Put`, `Delete`), it is flushed back to disk using `gob.NewEncoder` to ensure durability.
- **Buckets**: Keys are composited using a bucket prefix (`string(bucket) + "|" + string(key)`). The two primary buckets are `BlocksBucket` (for chain history) and `ChainStateBucket` (for the active UTXO index).

---

## The End-to-End Transaction Lifecycle

How do all these modules work together when a user sends funds?

1. **Initiation & Signing (`wallet` -> `core`)**: 
   A user wants to send DNY. The wallet selects unspent outputs from the `UTXOSet` that cover the amount plus the dynamic network fee. It creates a `Transaction` and uses its ECDSA private key to sign the hash of the inputs.

2. **Submission & Mempool Entry (`network`)**: 
   The transaction is submitted to the local node. The `Mempool` checks the database to confirm the referenced UTXOs exist and are unspent. It mathematically verifies the cryptographic signatures and fee limits. If valid, the transaction is added to the in-memory `map[string]*core.Transaction`.

3. **Network Broadcast (`network`)**: 
   The node wraps the transaction in a `TxMsg`, encodes it, and pushes it to the `incoming` channels of all connected peers (`n.SendToPeer`).

4. **Block Assembly (`main` -> `economics` -> `core`)**: 
   A miner runs an infinite loop. It creates a new `CoinbaseTx` paying the 50 DNY reward to its own address. It retrieves all valid pending transactions from the `Mempool`. It bundles them into a new `Block`, generating the `MerkleRoot` for the block header.

5. **Mining Consensus (`consensus`)**: 
   The miner passes the assembled block to the `PoUWEngine`. The engine enters a hot loop, incrementing the `Nonce` and hashing the header until a hash satisfying the `DifficultyTarget` is discovered.

6. **Commitment & State Transition (`core` -> `database`)**: 
   Once the hash is found, the block is valid. The `Blockchain.AddBlock()` method serializes the block and saves it to the `BlocksBucket`. It then calls `UTXOSet.Update()`, which iterates through the block's transactions, deleting spent inputs from the `ChainStateBucket` and adding the newly created outputs. 

7. **Eviction & Propagation (`main` -> `network`)**: 
   The node evicts the mined transactions from its mempool. Finally, it uses `BroadcastBlock` to distribute the winning block across the gossip network so peer nodes can validate it, append it to their local chains, and reset their own mining loops.