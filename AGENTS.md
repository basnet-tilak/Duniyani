# AI Agent Guidance for Duniyani

## Purpose
This file helps AI coding agents understand the Duniyani repository quickly and make productive, low-risk changes.

## Project overview
- Duniyani is a modular Go 1.22 blockchain node blueprint for a Layer 1 DePIN-style network.
- Core packages are separated by responsibility:
  - `types`: canonical domain models (`Block`, `Transaction`, `BlockHeader`, etc.)
  - `state`: blockchain state management and mempool handling
  - `network`: abstract P2P/gossip networking layer
  - `consensus`: pluggable consensus engine interface and mock engine
  - `core`: block validation, chain handling, and Merkle utilities
  - `crypto`: cryptographic primitives and signing/verification helpers
  - `database`: persistence abstractions
  - `economics`: economic rules and incentives logic
  - `wallet`: wallet key management and signing
- `main.go` wires the modules together and runs the node simulator.

## Build and test commands
Use the repository's documented commands first:
- `go mod tidy` to install dependencies
- `go run main.go` to run the node application
- `make test` to run the standard test suite
- `make test-race` to run tests with Go race detection
- `make bench` for benchmarks

Manual equivalents:
- `go test ./...`
- `go test -race ./...`
- `go test -bench=. -benchmem ./...`

## Agent behavior guidance
- Preserve modular boundaries and keep package responsibilities clear.
- When changing consensus behavior, update `consensus` only and avoid leaking consensus-specific logic into `core` or `network`.
- Prefer small, focused refactors over broad architectural rewrites.
- Follow existing Go conventions in the repo; do not introduce unrelated frameworks or large external dependencies.
- Use the repository README for guidance on commands and setup.

## When editing tests
- The codebase already expects the developer to install dependencies first.
- Keep concurrency-related changes race-safe and verify with `make test-race` when relevant.

## Notes for AI agents
- There is no existing `.github/copilot-instructions.md` or AGENTS file; use this file as the primary workspace guidance.
- Do not assume additional documentation exists beyond `README.md`.
- Focus on the existing package structure and the blueprint nature of the project.
