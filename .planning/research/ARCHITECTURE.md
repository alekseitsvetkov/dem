# Architecture Research: HLTV CLI

## Proposed Components

| Component | Responsibility |
|-----------|----------------|
| `cmd/hltv` | Application entry point and root command wiring. |
| `internal/cli` | Cobra commands, flag parsing, stdout/stderr handling, exit behavior. |
| `internal/hltv` | Provider interface and HLTV-specific implementation. |
| `internal/hltv/parser` | HTML parser functions for events, results, match demo links. |
| `internal/domain` | Typed data models returned by commands and encoded as JSON. |
| `internal/output` | JSON encoding, structured error formatting, stdout/stderr rules. |

## Data Flow

1. User invokes a command with flags or a match ID.
2. CLI validates input and builds a request object.
3. HLTV provider fetches the relevant public page using a configured HTTP client.
4. Parser extracts typed domain objects from HTML.
5. Output layer encodes the result as JSON to stdout.
6. Errors are encoded to stderr and produce non-zero exit codes.

## Extension Path

New commands should add:

- A Cobra command file under `internal/cli`.
- A provider method if new HLTV data is needed.
- A parser fixture test for any new page shape.
- A domain response type with JSON tags.

## Build Order Implications

Start with CLI skeleton and JSON/error contracts, then implement the provider and parser with fixtures. This keeps command behavior testable before relying on live HLTV pages.
