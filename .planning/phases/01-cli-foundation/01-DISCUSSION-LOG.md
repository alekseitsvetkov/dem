# Phase 1: CLI Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md - this log preserves the alternatives considered.

**Date:** 2026-05-02
**Phase:** 1-CLI Foundation
**Areas discussed:** JSON Shape, Error Contract, Command Surface, Extensibility Boundary, Baseline Command

---

## JSON Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Raw arrays/objects | Simple for v1, but harder to extend without breaking scripts. | |
| Consistent envelope | Use `data` and `meta` to preserve forward compatibility. | yes |

**User's choice:** User asked for best-practice defaults.
**Notes:** Best-practice decision is a consistent response envelope for successful JSON output.

---

## Error Contract

| Option | Description | Selected |
|--------|-------------|----------|
| Plain text stderr | Familiar for humans, less useful for automation. | |
| JSON stderr | Keeps scripts able to parse failures consistently. | yes |

**User's choice:** User asked for best-practice defaults.
**Notes:** Best-practice decision is JSON errors on stderr with stable machine-readable codes.

---

## Command Surface

| Option | Description | Selected |
|--------|-------------|----------|
| Top-level commands | `dem events`, `dem results`, `dem demo <match-id>`; direct and easy to script. | yes |
| Nested command groups | More taxonomy now, but unnecessary before the CLI grows. | |

**User's choice:** The user specified the CLI should be named `dem`.
**Notes:** Top-level command names preserve the v1 project language while keeping room to group future larger command families.

---

## Extensibility Boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Internal extensibility | Clean command registration, package boundaries, and interfaces. | yes |
| External plugin platform | More flexible, but premature for v1. | |

**User's choice:** User asked for best-practice defaults.
**Notes:** External plugins are deferred until the CLI has a stable contract and real extension pressure.

---

## Baseline Command

| Option | Description | Selected |
|--------|-------------|----------|
| `dem version` | Proves JSON output with build/version metadata if available. | yes |
| `dem info` | Proves JSON output without requiring build metadata. | yes |

**User's choice:** User asked for best-practice defaults.
**Notes:** Prefer `dem version` when build metadata is available; otherwise use `dem info`.

## the agent's Discretion

- Exact Cobra wiring and Go package layout are left to the planner.
- Exact initial version/info fields are left to the planner as long as stdout remains successful JSON only.

## Deferred Ideas

- External plugin support.
