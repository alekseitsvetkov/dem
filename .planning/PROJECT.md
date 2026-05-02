# HLTV CLI

## What This Is

HLTV CLI is an extensible Go command-line tool for retrieving Counter-Strike data from HLTV and emitting JSON for scripts, automations, and downstream tools. The first iteration focuses on three read-only workflows: listing Tier 1 events, listing completed match results, and returning the demo download link for a match by HLTV match ID.

The project should be easy to expand with new commands and providers without rewriting the core CLI contract.

## Core Value

Users can reliably fetch HLTV event, result, and demo-link data as stable JSON from a script-friendly CLI.

## Requirements

### Validated

(None yet - ship to validate)

### Active

- [ ] CLI is written in Go with an extensible command architecture.
- [ ] CLI emits JSON only on stdout for successful commands.
- [ ] User can list Tier 1 HLTV events.
- [ ] User can list completed match results from HLTV.
- [ ] User can request a demo download link by HLTV match ID.
- [ ] HLTV fetching and parsing are isolated behind replaceable internal interfaces.
- [ ] Network and parsing failures return structured errors and non-zero exit codes.

### Out of Scope

- Live match tracking - v1 focuses on events, completed results, and demo links.
- Downloading demo files - v1 returns the demo link only.
- Team/date demo search - v1 accepts a direct HLTV match ID for reliability.
- Non-JSON terminal tables - the CLI is intended for automation-first usage.
- User accounts, authentication, or private HLTV access - v1 uses public HLTV pages only.

## Context

The user wants an extensible CLI that can grow with new features over time. The target language is Go, which points toward a command tree with small internal packages and strongly typed domain models. The output contract is JSON-only, so every command should produce predictable machine-readable objects and arrays rather than terminal tables.

HLTV appears to expose the needed data through public pages such as results and match pages rather than an official stable API. That makes parser resilience a first-class design concern: selectors should be isolated, fixture-tested, and easy to update when HLTV markup changes.

Initial command shape:

- `dem events` - returns Tier 1 events.
- `dem results` - returns completed matches.
- `dem demo <match-id>` - returns a demo download link for the given HLTV match ID when available.

## Constraints

- **Language**: Go - selected by the user.
- **Output contract**: JSON only - stdout must remain machine-readable on success.
- **Input contract**: Demo lookup accepts an HLTV match ID - avoids fuzzy matching in v1.
- **Data source**: Public HLTV pages - no official API dependency assumed.
- **Extensibility**: Commands and HLTV parsing must be modular - future features should not require a rewrite.
- **Politeness**: HTTP fetching should use timeouts, clear user-agent, retries only where safe, and conservative request volume - scraping should be respectful and testable.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Build in Go | User selected Go; supports strong CLI binaries and maintainable internal packages. | - Pending |
| JSON-only output | User selected JSON; supports automation and downstream tools. | - Pending |
| Accept HLTV match ID for demo command | User selected match ID; avoids ambiguous search/disambiguation for v1. | - Pending |
| Keep HLTV access behind provider/parser interfaces | HLTV markup can change; isolation keeps future fixes scoped. | - Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `$gsd-transition`):
1. Requirements invalidated? Move to Out of Scope with reason
2. Requirements validated? Move to Validated with phase reference
3. New requirements emerged? Add to Active
4. Decisions to log? Add to Key Decisions
5. "What This Is" still accurate? Update if drifted

**After each milestone** (via `$gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check - still the right priority?
3. Audit Out of Scope - reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-02 after initialization*
