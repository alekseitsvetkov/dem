# Research Summary: HLTV CLI

## Stack

Build in Go with Cobra for subcommands, standard `net/http` for fetching, goquery for HTML parsing, and standard `encoding/json` for output. This keeps v1 simple while leaving a clean path to add commands and providers.

## Table Stakes

- JSON-only stdout for successful commands.
- Structured stderr errors with non-zero exit codes.
- Commands for Tier 1 events, completed results, and demo lookup by match ID.
- Typed response models with stable JSON fields.
- Fixture-tested HTML parsers isolated from command code.

## Watch Out For

- HLTV does not appear to offer a stable official API for these workflows, so public-page parsing is the fragile point.
- Tier 1 needs an explicit v1 definition before implementation.
- Demo lookup must distinguish "match not found", "demo not available", "HLTV unavailable", and "parser failed".

## Roadmap Implication

Implement the CLI shell and JSON contract first, then add HLTV fetching/parser infrastructure, then events/results parsing, then demo-link lookup. This order makes the extensibility goal real from the beginning.
