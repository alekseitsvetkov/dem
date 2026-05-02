<!-- GSD:project-start source:PROJECT.md -->
## Project

# HLTV CLI

HLTV CLI is an extensible Go command-line tool for retrieving Counter-Strike data from HLTV and emitting JSON for scripts, automations, and downstream tools. The first iteration focuses on three read-only workflows: listing Tier 1 events, listing completed match results, and returning the demo download link for a match by HLTV match ID.

Core value: Users can reliably fetch HLTV event, result, and demo-link data as stable JSON from a script-friendly CLI.
<!-- GSD:project-end -->

<!-- GSD:stack-start source:STACK.md -->
## Technology Stack

- Go for the CLI implementation.
- Cobra is the recommended command framework.
- Use standard `net/http` for fetching unless a stronger need appears.
- Use goquery for HLTV HTML parsing.
- Use standard `encoding/json` for stdout output.
- Use Go `testing`, `httptest`, fixture HTML, and golden output tests.
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

- Stdout is reserved for successful JSON only.
- Stderr is reserved for diagnostics and structured error output.
- Keep HLTV selectors and parsing rules out of command handlers.
- Prefer typed response structs with JSON tags over ad hoc maps.
- Fixture-test public-page parsing so HLTV markup changes are local to parser fixes.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Expected package shape:

- `cmd/dem` for the application entry point.
- `internal/cli` for commands, flags, and output wiring.
- `internal/hltv` for provider interfaces and public-page fetching.
- `internal/hltv/parser` for HTML parsing.
- `internal/domain` for response models.
- `internal/output` for JSON and error encoding.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project-specific skills found yet. Add skills to `.codex/skills/` when durable project knowledge emerges.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `$gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `$gsd-debug` for investigation and bug fixing
- `$gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `$gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` - do not edit manually.
<!-- GSD:profile-end -->
