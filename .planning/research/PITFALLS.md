# Pitfalls Research: HLTV CLI

## Pitfalls

| Pitfall | Warning Sign | Prevention Strategy | Phase |
|---------|--------------|---------------------|-------|
| HLTV markup changes break parsing | Tests only cover live requests or selectors are spread across commands. | Keep selectors in parser package and test against stored HTML fixtures. | Phase 2 |
| JSON schema drifts between commands | Commands encode ad hoc maps. | Define domain structs with JSON tags and golden tests for command output. | Phase 1 |
| Errors pollute stdout | Diagnostics printed with normal output. | Reserve stdout for successful JSON only; send errors to stderr. | Phase 1 |
| Fetching is too aggressive | Commands make many requests by default. | Add limits, timeouts, clear user-agent, and avoid recursive crawling. | Phase 2 |
| Tier 1 definition is ambiguous | Events command cannot explain why an event is included. | Define v1 Tier 1 as explicit HLTV-derived criteria and expose source fields in JSON. | Phase 3 |
| Demo unavailable cases are unclear | Missing demo link looks like a parser failure. | Return a distinct structured error for no demo available. | Phase 4 |

## Guidance

The riskiest part is not Go CLI mechanics; it is the public-page dependency. The implementation should assume selectors will change and make updates small, local, and covered by fixture tests.
