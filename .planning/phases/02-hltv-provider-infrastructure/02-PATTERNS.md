# Phase 2: HLTV Provider Infrastructure - Pattern Map

**Date:** 2026-05-02
**Status:** Complete

## Existing Patterns to Reuse

### Injectable IO Boundaries

- `internal/cli.NewRootCommand(out io.Writer, errOut io.Writer)` accepts writers rather than using globals.
- Phase 2 should mirror this style by accepting `*http.Client`, base URL, timeout, and user-agent in constructors.

### JSON/Error Boundary

- `internal/output.WriteJSON` and `internal/output.WriteErrorJSON` own encoding.
- Provider/parser packages should return typed values and typed errors. They should not write stdout, stderr, or JSON.

### Command Isolation

- `cmd/dem/main.go` only calls `os.Exit(cli.Execute())`.
- New provider code should not be imported by `cmd/dem`; command integration belongs in `internal/cli` in later phases.

### Test Style

- Existing tests use `bytes.Buffer`, direct function calls, and standard Go `testing`.
- Phase 2 tests should use `httptest.Server`, fake `RoundTripper`, `os.ReadFile` from `testdata`, and typed assertions.

## Planned File Roles

| File or Package | Role | Closest Existing Pattern |
|-----------------|------|--------------------------|
| `internal/hltv` | Public-page client, URL helpers, provider errors | `internal/cli` constructor/test injection style |
| `internal/hltv/parser` | HTML selectors and typed parsing | New package; keep isolated from CLI |
| `internal/domain` | Typed event/result/demo models with JSON tags | `internal/cli.versionInfo` JSON-tagged response shape |
| `internal/hltv/parser/testdata` | Sanitized parser fixtures | Go standard `testdata` convention |

## Integration Notes

- Future `dem events`, `dem results`, and `dem demo <match-id>` commands should depend on provider interfaces, not parser functions directly unless the plan explicitly wires a provider method.
- Parser tests should assert domain fields, not JSON strings.
- Error tests should assert stable codes: `network_error`, `http_error`, `parse_error`, and `unavailable_data`.
