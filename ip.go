package main

// ip.go
//
// This file handles the -ip flag.
//
// WHAT IT DOES:
//   Given an IP address, it returns the ISP name and geographic coordinates.
//
// HOW IT WORKS:
//   1. Validates the input is a real IP address using net.ParseIP. Rejects
//      garbage before it ever touches a URL.
//   2. Checks if the IP is private/reserved (127.x, 192.168.x, etc.)
//      Private IPs are never routed on the public internet, so no ISP owns them.
//      We classify them locally without making any network request.
//   3. For public IPs, sends a GET request to ip-api.com which returns JSON like:
//      {"status":"success","isp":"Google LLC","city":"Mountain View","lat":37.4,"lon":-122.07}
//   4. Decodes the JSON into the IPResponse struct using encoding/json.
//   5. Prints the result and saves it to a file.
//
// WHY ip-api.com?
//   It's free, requires no API key, and returns clean JSON. Perfect for OSINT.
//   It does rate-limit heavy usage, but for this project one request at a time is fine.
//   NOTE: the free tier is HTTP only (no TLS), so the response is unauthenticated
//   and unencrypted in transit. Treat ISP/geo data from this source as unverified.

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// IPResponse maps the JSON fields that ip-api.com returns.
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
func lookupIP(ip string) {
	if net.ParseIP(ip) == nil {
		fmt.Println("Error: invalid IP address")
		os.Exit(1)
	}

	var isp, city string
	var lat, lon float64

	if isPrivateIP(ip) {
		isp = classifyPrivateIP(ip)
		city = "Private/Reserved Address"
		lat = 0.0
		lon = 0.0
	} else {
		url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
		resp, err := getWithAgent(url)
		if err != nil {
			fmt.Printf("Error connecting to IP lookup service: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			os.Exit(1)
		}

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
			isp = fmt.Sprintf("Unknown (%s)", data.Message)
			city = "Unknown"
		}
	}

	fmt.Printf("ISP: %s\n", isp)
	fmt.Printf("City Lat/Lon:\t(%.4f) / (%.4f)\n", lat, lon)

	filename := getOutputFilename()
	content := fmt.Sprintf(
		"IP: %s\nISP: %s\nCity: %s\nCity Lat/Lon: (%.4f) / (%.4f)\n",
		ip, isp, city, lat, lon,
	)
	saveResult(content, filename)
}

// isPrivateIP returns true if the IP belongs to a private or reserved range.
//
//   - RFC 1918: 10.x.x.x, 172.16-31.x.x, 192.168.x.x (private LAN)
//   - RFC 5735: 127.x.x.x (loopback / localhost)
//   - RFC 3927: 169.254.x.x (link-local / APIPA)
func isPrivateIP(ip string) bool {
	if strings.HasPrefix(ip, "127.") ||
		strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "169.254.") ||
		ip == "::1" {
		return true
	}
	if strings.HasPrefix(ip, "172.") {
		parts := strings.Split(ip, ".")
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n >= 16 && n <= 31 {
				return true
			}
		}
	}
	return false
}

// classifyPrivateIP returns a descriptive explanation for why a private IP
// has no ISP.
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
