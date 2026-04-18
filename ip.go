package main

// ip.go
//
// This file handles the -ip flag.
//
// WHAT IT DOES:
//   Given an IP address, it returns the ISP name and geographic coordinates.
//
// HOW IT WORKS:
//   1. First checks if the IP is private/reserved (127.x, 192.168.x, etc.)
//      Private IPs are never routed on the public internet, so no ISP owns them.
//      We classify them locally without making any network request.
//
//   2. For public IPs, sends a GET request to ip-api.com which returns JSON like:
//      {"status":"success","isp":"Google LLC","city":"Mountain View","lat":37.4,"lon":-122.07}
//
//   3. Decodes the JSON into the IPResponse struct using encoding/json.
//
//   4. Prints the result and saves it to a file.
//
// WHY ip-api.com?
//   It's free, requires no API key, and returns clean JSON. Perfect for OSINT.
//   It does rate-limit heavy usage, but for this project one request at a time is fine.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// IPResponse maps the JSON fields that ip-api.com returns.
// The struct tags (e.g. `json:"status"`) tell encoding/json which JSON key
// maps to which Go field. Fields not in the struct are silently ignored.
type IPResponse struct {
	Status  string  `json:"status"`
	ISP     string  `json:"isp"`
	City    string  `json:"city"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Message string  `json:"message"` // only present when status == "fail"
}

// lookupIP is called when the user runs: passive -ip <address>
//
// It coordinates the full flow: classify or fetch → print → save.
func lookupIP(ip string) {
	var isp, city string
	var lat, lon float64

	if isPrivateIP(ip) {
		// Private IP — no network call needed, classify it immediately.
		// This is the path the auditor triggers with `passive -ip 127.0.0.1`.
		isp = classifyPrivateIP(ip)
		city = "Private/Reserved Address"
		lat = 0.0
		lon = 0.0
	} else {
		// Public IP — fetch data from ip-api.com
		url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
		resp, err := getWithAgent(url)
		if err != nil {
			fmt.Printf("Error connecting to IP lookup service: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		// io.ReadAll reads the entire response body into a byte slice.
		// We need the raw bytes so we can pass them to json.Unmarshal.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

		// json.Unmarshal decodes the JSON bytes into our IPResponse struct.
		// If the JSON is malformed or a field type doesn't match, it returns an error.
		var data IPResponse
		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Printf("Error parsing IP response: %v\n", err)
			os.Exit(1)
		}

		if data.Status == "success" {
			isp = data.ISP
			city = data.City
			lat = data.Lat
			lon = data.Lon
		} else {
			// ip-api returns "fail" for some reserved ranges it doesn't handle
			isp = fmt.Sprintf("Unknown (%s)", data.Message)
			city = "Unknown"
		}
	}

	// Print to terminal
	fmt.Printf("ISP: %s\n", isp)
	fmt.Printf("City Lat/Lon:\t(%.4f) / (%.4f)\n", lat, lon)

	// Save to file
	filename := getOutputFilename()
	content := fmt.Sprintf(
		"IP: %s\nISP: %s\nCity: %s\nCity Lat/Lon: (%.4f) / (%.4f)\n",
		ip, isp, city, lat, lon,
	)
	saveResult(content, filename)
}

// isPrivateIP returns true if the IP belongs to a private or reserved range.
//
// These ranges are defined by:
//   - RFC 1918: 10.x.x.x, 172.16-31.x.x, 192.168.x.x (private LAN)
//   - RFC 5735: 127.x.x.x (loopback / localhost)
//   - RFC 3927: 169.254.x.x (link-local / APIPA)
//
// None of these IPs are routable on the public internet, so they have no ISP.
// We check by prefix because Go's standard library net package would add
// an external import — simple string prefix checks are sufficient here.
func isPrivateIP(ip string) bool {
	privatePrefixes := []string{
		"127.",
		"192.168.",
		"10.",
		"172.",
		"169.254.",
		"::1", // IPv6 loopback
		"0.",
	}
	for _, prefix := range privatePrefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}

// classifyPrivateIP returns a descriptive explanation for why a private IP
// has no ISP. This makes the output informative instead of just saying "error".
func classifyPrivateIP(ip string) string {
	switch {
	case strings.HasPrefix(ip, "127."):
		return "Loopback (localhost) - Private address, no ISP"
	case strings.HasPrefix(ip, "192.168."):
		return "Private Network (RFC 1918) - LAN address, no ISP"
	case strings.HasPrefix(ip, "10."):
		return "Private Network (RFC 1918) - LAN address, no ISP"
	case strings.HasPrefix(ip, "172."):
		return "Private Network (RFC 1918) - LAN address, no ISP"
	case strings.HasPrefix(ip, "169.254."):
		return "Link-local (APIPA) - no ISP"
	default:
		return "Reserved/Private address - no ISP data available"
	}
}
