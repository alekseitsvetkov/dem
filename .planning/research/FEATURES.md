# Feature Research: HLTV CLI

## Table Stakes for v1

- JSON-only command output with a documented schema per command.
- Clear command separation: events, results, demo lookup.
- Stable identifiers in output where available: event ID, match ID, team names, event name, date, map/format metadata.
- Limit/filter flags for list endpoints so commands do not emit unbounded results.
- Structured errors with non-zero exit codes when HLTV is unreachable, markup is unparseable, or a demo is unavailable.

## Differentiators for Later

- Team/date search for demo lookup.
- Event filters by year, status, prize pool, location, or HLTV tier.
- Result filters by event, team, date range, map, best-of format, or star rating.
- Demo download command that fetches the file after returning/confirming the link.
- Cache layer for repeated requests.
- Alternate providers if HLTV access changes.

## Anti-Features for v1

- Terminal tables: conflicts with the JSON-only contract.
- Browser automation as the default path: heavier, harder to ship as a simple CLI.
- High-volume crawling: unnecessary for the first workflows and more likely to break or be blocked.
- Fuzzy demo search: useful later, but less reliable than direct match ID lookup.

## v1 Command Sketch

```text
dem events --tier 1 --limit 50
dem results --limit 100
dem demo <match-id>
```

## Output Sketch

```json
{
  "matchId": 123456,
  "demoUrl": "https://...",
  "sourceUrl": "https://www.hltv.org/matches/123456/..."
}
```
