package hltv

import (
	"strconv"
	"strings"
)

const DefaultBaseURL = "https://www.hltv.org"

type URLs struct {
	BaseURL string
}

func NewURLs(baseURL string) URLs {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return URLs{BaseURL: strings.TrimSuffix(baseURL, "/")}
}

func (u URLs) EventsURL() string {
	return u.BaseURL + "/events"
}

func (u URLs) ResultsURL() string {
	return u.BaseURL + "/results"
}

func (u URLs) MatchURL(matchID int) string {
	return u.BaseURL + "/matches/" + strconv.Itoa(matchID) + "/-"
}
