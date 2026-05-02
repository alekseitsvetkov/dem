# Requirements: HLTV CLI

**Defined:** 2026-05-02
**Core Value:** Users can reliably fetch HLTV event, result, and demo-link data as stable JSON from a script-friendly CLI.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### CLI Foundation

- [ ] **CLI-01**: User can run the compiled Go CLI and see JSON output for successful commands.
- [ ] **CLI-02**: User can request command help without triggering HLTV network calls.
- [ ] **CLI-03**: User receives structured JSON-compatible error details on stderr and a non-zero exit code when a command fails.
- [ ] **CLI-04**: Future commands can be added without changing existing command handlers or JSON response types.

### HLTV Access

- [ ] **HLTV-01**: CLI fetches public HLTV pages with timeouts and a configured user-agent.
- [ ] **HLTV-02**: HLTV provider logic is isolated behind interfaces so tests can use fixture HTML or mock transports.
- [ ] **HLTV-03**: Parser tests cover expected HLTV page structures for events, results, and match demo links.

### Events

- [ ] **EVNT-01**: User can list Tier 1 HLTV events as JSON.
- [x] **EVNT-02**: Each event includes stable available fields such as event ID, name, date range, location, and source URL.
- [ ] **EVNT-03**: User can limit the number of events returned.

### Results

- [ ] **RSLT-01**: User can list completed HLTV match results as JSON.
- [x] **RSLT-02**: Each result includes stable available fields such as match ID, teams, score, event, date, format, and source URL.
- [ ] **RSLT-03**: User can limit the number of results returned.

### Demo Lookup

- [ ] **DEMO-01**: User can pass an HLTV match ID and receive a JSON response containing the match source URL.
- [ ] **DEMO-02**: User receives a demo download URL when HLTV exposes one for the match.
- [ ] **DEMO-03**: User receives a distinct structured error when the match exists but no demo link is available.

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Discovery

- **DISC-01**: User can search for a match by teams and date.
- **DISC-02**: User can filter results by team, event, date range, or map.
- **DISC-03**: User can filter events by year, status, or location.

### Downloads

- **DOWN-01**: User can download a demo file directly from the CLI.
- **DOWN-02**: User can choose an output directory for downloaded demos.

### Performance

- **PERF-01**: User can enable local caching for repeated requests.

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Live match tracking | v1 focuses on completed and static HLTV data. |
| Demo file download | User asked for a download link, not file transfer. |
| Fuzzy match search | User selected direct HLTV match ID for v1. |
| Human-readable tables | User selected JSON-only output. |
| Private/authenticated HLTV access | v1 should use public pages only. |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLI-01 | Phase 1 | Pending |
| CLI-02 | Phase 1 | Pending |
| CLI-03 | Phase 1 | Pending |
| CLI-04 | Phase 1 | Pending |
| HLTV-01 | Phase 2 | Pending |
| HLTV-02 | Phase 2 | Pending |
| HLTV-03 | Phase 2 | Pending |
| EVNT-01 | Phase 3 | Pending |
| EVNT-02 | Phase 3 | Completed (Plan 03-01) |
| EVNT-03 | Phase 3 | Pending |
| RSLT-01 | Phase 3 | Pending |
| RSLT-02 | Phase 3 | Completed (Plan 03-01) |
| RSLT-03 | Phase 3 | Pending |
| DEMO-01 | Phase 4 | Pending |
| DEMO-02 | Phase 4 | Pending |
| DEMO-03 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0

---
*Requirements defined: 2026-05-02*
*Last updated: 2026-05-02 after roadmap creation*
