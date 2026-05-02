# Roadmap: HLTV CLI

## Overview

Build the CLI from the inside out: first establish the Go command skeleton and JSON/error contract, then add resilient HLTV fetching and parsing infrastructure, then implement event/result list commands, and finally add match-demo lookup by HLTV match ID. Each phase leaves behind a working, testable layer that future commands can reuse.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: CLI Foundation** - Create the Go CLI skeleton, command architecture, JSON output, and structured error contract.
- [ ] **Phase 2: HLTV Provider Infrastructure** - Add HTTP fetching, provider interfaces, parser boundaries, and fixture-based tests.
- [ ] **Phase 3: Events and Results Commands** - Implement Tier 1 events and completed results JSON commands.
- [ ] **Phase 4: Demo Link Lookup** - Implement demo URL lookup by HLTV match ID with precise unavailable/error behavior.

## Phase Details

### Phase 1: CLI Foundation
**Goal**: Deliver a working Go CLI foundation that can grow with new commands and always preserves JSON stdout.
**Depends on**: Nothing (first phase)
**Requirements**: [CLI-01, CLI-02, CLI-03, CLI-04]
**Success Criteria** (what must be TRUE):
  1. User can run the CLI binary and receive valid JSON from at least one non-network command.
  2. User can inspect command help without triggering HLTV requests.
  3. Failed commands write structured error details to stderr and return non-zero exit codes.
  4. A new command can be added through the command registration pattern without touching existing handlers.
**Plans**: 2 plans

Plans:
- [ ] 01-01: Scaffold Go module, root command, command registration, and JSON output helpers.
- [ ] 01-02: Add structured error handling, help behavior, and CLI contract tests.

### Phase 2: HLTV Provider Infrastructure
**Goal**: Build the reusable HLTV access layer with polite HTTP behavior and fixture-tested parsing boundaries.
**Depends on**: Phase 1
**Requirements**: [HLTV-01, HLTV-02, HLTV-03]
**Success Criteria** (what must be TRUE):
  1. HLTV requests use configured timeout and user-agent behavior.
  2. Provider methods can be tested without live network calls.
  3. Parser tests use stored HTML fixtures for all v1 page types.
  4. Network, parse, and unavailable-data failures are distinguishable.
**Plans**: 2 plans

Plans:
- [ ] 02-01: Implement HTTP client/provider interfaces and test doubles.
- [ ] 02-02: Implement parser package structure, fixtures, and parser error taxonomy.

### Phase 3: Events and Results Commands
**Goal**: Expose JSON commands for Tier 1 events and completed HLTV match results.
**Depends on**: Phase 2
**Requirements**: [EVNT-01, EVNT-02, EVNT-03, RSLT-01, RSLT-02, RSLT-03]
**Success Criteria** (what must be TRUE):
  1. User can run `dem events --tier 1` and receive a JSON array of Tier 1 events.
  2. Event JSON includes available ID, name, date range, location, and source URL fields.
  3. User can run `dem results` and receive a JSON array of completed matches.
  4. Result JSON includes available match ID, teams, score, event, date, format, and source URL fields.
  5. List commands support a limit flag that bounds returned records.
**Plans**: 3 plans

Plans:
- [ ] 03-01: Define event/result domain models and JSON schemas.
- [ ] 03-02: Implement Tier 1 events parser/provider/command.
- [ ] 03-03: Implement completed results parser/provider/command.

### Phase 4: Demo Link Lookup
**Goal**: Let users retrieve a demo download link for a specific HLTV match ID.
**Depends on**: Phase 3
**Requirements**: [DEMO-01, DEMO-02, DEMO-03]
**Success Criteria** (what must be TRUE):
  1. User can run `dem demo <match-id>` and receive JSON containing the match source URL.
  2. When HLTV exposes a demo link, the command returns the direct or canonical demo URL.
  3. When the match has no demo link, the command returns a distinct structured unavailable-demo error.
  4. Invalid match IDs fail before network access with a structured validation error.
**Plans**: 2 plans

Plans:
- [ ] 04-01: Implement match page fetching/parsing for demo links by match ID.
- [ ] 04-02: Add demo command validation, JSON output, and unavailable-demo tests.

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. CLI Foundation | 2/2 | Security review pending | - |
| 2. HLTV Provider Infrastructure | 0/2 | Not started | - |
| 3. Events and Results Commands | 0/3 | Not started | - |
| 4. Demo Link Lookup | 0/2 | Not started | - |
