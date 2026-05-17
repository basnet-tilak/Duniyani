# Duniyani: A Modular Layer 1 Blockchain Node Blueprint

## 1. Overview

Duniyani is a production-ready, modular codebase blueprint for a high-performance Layer 1 blockchain client, specifically designed for Decentralized Physical Infrastructure Networks (DePIN). It is written in modern Go (1.22+) and follows 2026-era standards for Web3 applications.

The architecture strictly separates core concerns into distinct, pluggable modules:

-   `/types`: Defines the core data structures like `Block`, `Transaction`, and `BlockHeader`.
-   `/state`: Manages the blockchain's state, including a highly concurrent `Mempool` for pending transactions.
-   `/network`: Provides an abstracted P2P networking layer, simulating a `libp2p` gossip network for broadcasting blocks and transactions.
-   `/consensus`: Defines a pluggable consensus engine interface, with a mock `PoUWEngine` (Proof of Useful Work) as a placeholder.
-   `main.go`: The orchestrator that wires all the modules together, initializes the node, and handles graceful startup and shutdown.

## 2. Getting Started

### Prerequisites

-   **Go 1.22+**: You must have a recent version of Go installed and configured on your system.

### Installation & Setup

**This is the most critical step and the likely reason your tests are not running.** Before you can run the node or its tests, you must download the necessary dependencies, such as the `testify` package used in our test suite.

From the root of the project directory, run the following command:

```bash
go mod tidy
```

This command will read the `go.mod` file, find all the required dependencies, and download them to your local Go module cache.

## 3. How to Run the Node

Once the dependencies are installed, you can run the main node application:

The application supports several modes of operation via command-line flags.

**1. Generate a Wallet:**
Before mining or receiving funds, you need a wallet address.
```bash
go run main.go -wallet
```

**2. Run the Full Node Services:**
To start the networking node and actively participate in the network (without mining):
```bash
go run main.go -node
```

**3. Run the Node and Mine:**
To start the node and enable the Proof of Useful Work (PoUW) miner, provide your wallet address:
```bash
go run main.go -node -miner <your_wallet_address>
```

You will see log output indicating that the node has started, is simulating transactions, and is creating new blocks every 10 seconds.

## 4. How to Run the Test Suite

The project includes a comprehensive, enterprise-grade test suite. We have provided a `Makefile` to simplify the process of running these tests.

> **Note:** If these commands fail, it is almost certainly because the dependencies were not installed. Please run `go mod tidy` first.

### Using the Makefile (Recommended)

The `Makefile` provides several targets for building and testing the project.

**To run all standard unit and integration tests:**

```bash
make test
```

**To run tests with the Race Detector:**

This is one of the most powerful features of Go's toolchain. It will detect if there are any race conditions in the code, which is critical for a concurrent application like a blockchain node. Our test suite is designed to be run with this.

```bash
make test-race
```

**To run performance benchmarks:**

This will measure the performance of critical functions, such as hashing and block validation, and report on memory allocations.

```bash
make bench
```

For more advanced performance analysis, you can use tools like `benchstat` to compare benchmark results before and after code changes.

### Running `go test` Manually

If you prefer not to use `make`, you can run the `go test` commands directly:

-   **Standard Tests**: `go test ./...`
-   **Race Detector**: `go test -race ./...`
-   **Benchmarks**: `go test -bench=. -benchmem ./...`
