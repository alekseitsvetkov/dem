# Stack Research: HLTV CLI

## Recommendation

Use Go with a small Cobra command tree, a typed internal domain layer, and an HLTV provider package that uses `net/http` plus HTML parsing through goquery. Keep stdout JSON-only and send diagnostics to stderr.

## Suggested Stack

| Area | Choice | Rationale | Confidence |
|------|--------|-----------|------------|
| CLI framework | `spf13/cobra` | Mature Go command framework used by large CLIs; clean subcommand expansion path. | High |
| HTTP client | Go `net/http` initially | Standard library is enough for GET requests, timeouts, user-agent, and testable transports. | High |
| HTML parsing | `PuerkitoBio/goquery` | CSS-selector querying over `net/html`; well suited to scraping public HLTV pages. | Medium |
| JSON | Go `encoding/json` | Standard library support is enough for stable machine output. | High |
| Testing | Go `testing`, fixture HTML files, `httptest` | Supports parser tests without hitting HLTV during test runs. | High |
| Build/release | GoReleaser later | Useful once binaries need cross-platform distribution; not required for v1. | Medium |

## Notes

- HLTV results are available through public results pages.
- HLTV upcoming/current match pages are public and can be used as references for URL and selector shape.
- Avoid a third-party unofficial HLTV API dependency for v1 unless it is clearly maintained; direct provider code gives better control over JSON shape and error behavior.

## Sources Checked

- HLTV results page: https://www.hltv.org/results
- HLTV matches page: https://www.hltv.org/matches
- Cobra repository: https://github.com/spf13/cobra
- goquery package: https://pkg.go.dev/github.com/PuerkitoBio/goquery
