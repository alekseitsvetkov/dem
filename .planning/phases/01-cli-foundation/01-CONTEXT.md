# Phase 1: CLI Foundation - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the foundation for the `dem` Go CLI: binary naming, root command structure, JSON-only success output, structured error behavior, and an internal extension pattern that later HLTV commands can reuse. It does not fetch or parse HLTV pages yet; network/provider work begins in Phase 2.

</domain>

<decisions>
## Implementation Decisions

### Binary and Command Surface
- **D-01:** The CLI binary name is `dem`.
- **D-02:** Use simple top-level v1 commands: `dem events`, `dem results`, and `dem demo <match-id>`.
- **D-03:** Do not introduce nested command groups such as `dem matches results` in v1. The command tree can grow later if a family becomes large enough to justify grouping.
- **D-04:** Help/version commands must be local-only and must not trigger HLTV network calls.

### JSON Success Contract
- **D-05:** Successful command output goes to stdout as JSON only.
- **D-06:** Use a consistent response envelope for forward compatibility:

```json
{
  "data": {},
  "meta": {}
}
```

- **D-07:** Commands returning lists should place arrays under `data`, and optional pagination/request details under `meta`.
- **D-08:** Avoid raw top-level arrays because they are harder to extend without breaking scripts.

### Error Contract
- **D-09:** Failed commands write JSON to stderr and return a non-zero exit code.
- **D-10:** Error output should use a stable envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "human readable summary",
    "details": {}
  }
}
```

- **D-11:** Error codes should be machine-stable and snake_case, with messages kept human-readable but not relied upon by scripts.
- **D-12:** Validation errors should happen before any network access.

### Extensibility Boundary
- **D-13:** Phase 1 should optimize for internal extensibility: command registration, typed handlers, output helpers, and package boundaries that make new commands cheap to add.
- **D-14:** Do not build an external plugin system in v1. External plugins are deferred until there is real demand and a stable command/data contract.
- **D-15:** Future command handlers should depend on interfaces and output helpers, not on global stdout/stderr or direct process exits.

### Baseline Command
- **D-16:** Add a non-network command such as `dem version` or `dem info` to prove the JSON contract before HLTV fetching exists.
- **D-17:** Prefer `dem version` if the implementation has build metadata available; otherwise `dem info` may be used as a simple local contract check.

### the agent's Discretion
- The user explicitly asked for best-practice defaults where they were unsure. Downstream agents may choose the exact Go package layout and Cobra wiring details as long as they preserve the decisions above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Scope
- `.planning/PROJECT.md` - Project identity, core value, constraints, and locked top-level decisions.
- `.planning/REQUIREMENTS.md` - Phase 1 requirements `CLI-01` through `CLI-04`.
- `.planning/ROADMAP.md` - Phase 1 boundary, success criteria, and plan split.
- `.planning/research/STACK.md` - Recommended Go CLI stack.
- `.planning/research/ARCHITECTURE.md` - Proposed package boundaries and data flow.
- `AGENTS.md` - Project conventions and workflow guidance.

### External References
- Cobra documentation/repository - CLI command framework reference: https://github.com/spf13/cobra

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None yet. The repository currently contains planning artifacts only.

### Established Patterns
- No application code exists yet.
- Planning docs establish Go, JSON-only output, and isolated provider/parser boundaries.

### Integration Points
- Phase 1 creates the application entry point and internal CLI packages that later phases will connect HLTV provider logic into.

</code_context>

<specifics>
## Specific Ideas

- The CLI should be named `dem`.
- User prefers best-practice defaults for unresolved CLI foundation decisions.
- Keep v1 practical and script-friendly rather than overbuilding a plugin platform.

</specifics>

<deferred>
## Deferred Ideas

- External plugin support - defer until the CLI has stable command contracts and a real need for third-party extensions.

</deferred>

---

*Phase: 1-CLI Foundation*
*Context gathered: 2026-05-02*
