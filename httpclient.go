package main

// httpclient.go
//
// This file sets up a single shared HTTP client that every other file uses.
// Having one client (instead of creating a new one per request) is good practice
// in Go — it reuses TCP connections, handles timeouts consistently, and keeps
// the rest of the code clean.

import (
	"net/http"
	"time"
)

// client is the single shared HTTP client for the entire program.
//
// Two important settings:
//
//  1. Timeout: 10 seconds — if a server doesn't respond in 10 seconds,
//     the request fails with an error instead of hanging forever.
//     Without this, `passive -u "@someone"` could hang on a slow platform
//     and never finish.
//
//  2. CheckRedirect: stops automatic redirect following.
//     When you visit a URL that doesn't exist, many platforms (Facebook,
//     Instagram, Twitter) return HTTP 301 or 302 and redirect you to a
//     login page or a "not found" page. If we follow the redirect blindly,
//     we'd land on a 200 OK login page and wrongly report the account exists.
//     By returning http.ErrUseLastResponse we stop at the first response
//     and read the status code directly.
var client = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// getWithAgent sends a GET request to the given URL with a realistic browser
// User-Agent header.
//
// Why do we need a User-Agent?
// Many websites check the User-Agent and block requests that look like bots.
// Go's default User-Agent is "Go-http-client/1.1" — that gets blocked
// almost everywhere. Using a browser-like string gets us past basic bot filters.
//
// The Accept header tells the server we can handle HTML, which some sites
// require before they'll return a real page instead of an empty response.
func getWithAgent(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36",
	)
	req.Header.Set(
		"Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	)
	return client.Do(req)
}
