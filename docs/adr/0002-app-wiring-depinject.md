# 2. Application Wiring via Depinject

- **Status**: Proposed
- **Deciders**: Core Protocol Team, Lead Architect
- **Date**: 2026-09-09

## Context
The legacy Cosmos SDK `app.go` pattern relies on manual, imperative initialization of keepers and modules. This approach has several drawbacks:
1. High risk of hidden cyclic dependencies between keepers.
2. Poor testability in isolation (requires bootstrapping the entire app).
3. Verbose and error-prone boilerplate code.

Alternatively, `depinject` (Cosmos SDK v0.54+) provides declarative dependency injection. Conflicts and missing dependencies are caught at compile/build time rather than runtime.

## Decision
We will adopt `depinject` for application wiring from the very beginning of Phase 1 (ChaosChain L1 Core). We will not start with legacy `app.go` and migrate later, as this accumulates technical debt and makes the migration painful.

## Consequences
- **Positive**: Compile-time safety for dependencies, cleaner `app.go`, easier unit testing of isolated modules, alignment with modern Cosmos SDK best practices.
- **Negative**: Steeper learning curve for developers unfamiliar with `depinject` patterns.
- **Fallback**: If rejected, a hard deadline for migration to `depinject` must be set before Phase 1 Mainnet release, not left as an open-ended "later".