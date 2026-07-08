package main

// username.go
//
// This file handles the -u flag.
//
// WHAT IT DOES:
//   Given a username (with or without a leading @), checks whether that
//   username exists on 7 social platforms.
//
// HOW IT WORKS:
//   For each platform, we construct the profile URL and make a GET request,
//   then read the HTTP status code:
//
//     200      → profile likely exists → "yes"
//     404      → profile doesn't exist → "no"
//     301/302  → redirect, most platforms send non-existent users to a
//                login/error page → "no"
//     403/429  → blocked / rate limited, can't confirm → "blocked"
//
// The username is URL-escaped before being placed in any request path.
// Without escaping, characters like ?, #, & let user input alter the
// request path or inject query parameters into the outbound request.

import (
	"fmt"
	"net/url"
	"strings"
)

// Platform holds a social network name and the URL template for a profile.
type Platform struct {
	Name string
	URL  string
}

// checkUsername is called when the user runs: passive -u <username>
func checkUsername(username string) {
	username = strings.TrimPrefix(username, "@")
	safeUser := url.PathEscape(username)

	// Reddit uses the /about.json endpoint because the HTML profile page
	// returns 200 even for deleted accounts — the JSON endpoint returns 404
	// for non-existent users, which is more reliable.
	platforms := []Platform{
		{"Facebook", fmt.Sprintf("https://www.facebook.com/%s", safeUser)},
		{"Twitter", fmt.Sprintf("https://twitter.com/%s", safeUser)},
		{"Instagram", fmt.Sprintf("https://www.instagram.com/%s/", safeUser)},
		{"GitHub", fmt.Sprintf("https://github.com/%s", safeUser)},
		{"LinkedIn", fmt.Sprintf("https://www.linkedin.com/in/%s", safeUser)},
		{"TikTok", fmt.Sprintf("https://www.tiktok.com/@%s", safeUser)},
		{"Reddit", fmt.Sprintf("https://www.reddit.com/user/%s/about.json", safeUser)},
	}

	results := make(map[string]string)

	for _, p := range platforms {
		resp, err := getWithAgent(p.URL)
		if err != nil {
			results[p.Name] = "error"
			continue
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case 200:
			results[p.Name] = "yes"
		case 404:
			results[p.Name] = "no"
		case 301, 302:
			results[p.Name] = "no"
		case 403, 429:
			results[p.Name] = "blocked"
		default:
			results[p.Name] = "no"
		}
	}

	var sb strings.Builder
	for _, p := range platforms {
		status := results[p.Name]
		fmt.Printf("%s : %s\n", p.Name, status)
		sb.WriteString(fmt.Sprintf("%s : %s\n", p.Name, status))
	}

	filename := getOutputFilename()
	saveResult(sb.String(), filename)
}
