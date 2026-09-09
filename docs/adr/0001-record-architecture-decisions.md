# 1. Record Architecture Decisions

- **Status**: Accepted
- **Deciders**: Core Protocol Team
- **Date**: 2026-09-09

## Context
As ChaosChain evolves, architectural decisions need to be recorded to provide context for future developers and maintainers. Without a formal record, the rationale behind critical choices (e.g., consensus parameters, module wiring, cryptographic primitives) is lost, leading to architectural drift and repeated debates.

## Decision
We will use Architecture Decision Records (ADRs) based on the Michael Nygard template for any change that affects:
- Public interfaces of modules.
- Application wiring (`app.go` dependency injection).
- Protocol invariants (e.g., BFT safety bounds, tokenomics formulas).
- Cryptographic primitives or security models.

## Consequences
- **Positive**: Clear historical context, easier onboarding, reduced architectural debt.
- **Negative**: Slight overhead in the development process to draft and review ADRs.
- **Mitigation**: ADRs are mandatory for the "Definition of Done" of any epic or major feature.