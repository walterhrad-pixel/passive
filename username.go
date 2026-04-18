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
//   For each platform, we construct the profile URL and make a GET request.
//   We then read the HTTP status code of the response:
//
//     200  → The page loaded successfully → profile likely exists → "yes"
//     404  → Page not found → profile doesn't exist → "no"
//     301/302 → Redirect → platforms redirect non-existent users to login
//               or error pages, so a redirect means the profile is NOT there → "no"
//     403  → The platform blocked our request (bot protection) → "blocked"
//     429  → Rate limited (too many requests) → "blocked"
//
// WHY STATUS CODES AND NOT PAGE CONTENT?
//   Reading the full page body and searching for specific text would be more
//   accurate but also fragile — platforms change their HTML constantly.
//   Status codes are part of the HTTP standard and much more stable.
//
// LIMITATIONS:
//   Facebook and LinkedIn aggressively block automated requests.
//   They may show "blocked" even for real accounts. This is expected
//   and worth mentioning during the audit.

import (
	"fmt"
	"strings"
)

// Platform holds a social network name and the URL template for a profile.
type Platform struct {
	Name string
	URL  string
}

// checkUsername is called when the user runs: passive -u <username>
func checkUsername(username string) {
	// Strip the leading @ character if present.
	// The auditor will run `passive -u "@user01"` — we need to strip "@"
	// before building the URLs, otherwise we'd request /@@user01.
	username = strings.TrimPrefix(username, "@")

	// The 7 platforms we check. Each URL is the standard public profile path.
	// Reddit uses the /about.json endpoint because the HTML profile page
	// returns 200 even for deleted accounts — the JSON endpoint returns 404
	// for non-existent users, which is more reliable.
	platforms := []Platform{
		{"Facebook", fmt.Sprintf("https://www.facebook.com/%s", username)},
		{"Twitter", fmt.Sprintf("https://twitter.com/%s", username)},
		{"Instagram", fmt.Sprintf("https://www.instagram.com/%s/", username)},
		{"GitHub", fmt.Sprintf("https://github.com/%s", username)},
		{"LinkedIn", fmt.Sprintf("https://www.linkedin.com/in/%s", username)},
		{"TikTok", fmt.Sprintf("https://www.tiktok.com/@%s", username)},
		{"Reddit", fmt.Sprintf("https://www.reddit.com/user/%s/about.json", username)},
	}

	// results stores the status for each platform.
	// We use a map here so we can look up results by name when printing.
	results := make(map[string]string)

	for _, p := range platforms {
		resp, err := getWithAgent(p.URL)
		if err != nil {
			// Network error (DNS failure, timeout, etc.)
			results[p.Name] = "error"
			continue
		}
		// We must close the body even if we don't read it.
		// Not closing causes resource/file descriptor leaks.
		resp.Body.Close()

		switch resp.StatusCode {
		case 200:
			results[p.Name] = "yes"
		case 404:
			results[p.Name] = "no"
		case 301, 302:
			// Redirect — platform is sending us away from this profile URL.
			// For non-existent users, this is how most platforms say "not found"
			// without returning a 404.
			results[p.Name] = "no"
		case 403, 429:
			// 403 = Forbidden (bot blocked)
			// 429 = Too Many Requests (rate limited)
			// In both cases the account might exist, we just can't confirm it.
			results[p.Name] = "blocked"
		default:
			results[p.Name] = "no"
		}
	}

	// Print results and build file content in the same order as the platforms
	// slice (not the map order, which is random in Go).
	var sb strings.Builder
	for _, p := range platforms {
		status := results[p.Name]
		fmt.Printf("%s : %s\n", p.Name, status)
		sb.WriteString(fmt.Sprintf("%s : %s\n", p.Name, status))
	}

	filename := getOutputFilename()
	saveResult(sb.String(), filename)
}
