package main

// fullname.go
//
// This file handles the -fn flag.
//
// WHAT IT DOES:
//   Given a full name like "Jean Dupont", searches public French phone
//   directories for the person's address and phone number.
//
// HOW IT WORKS:
//   1. Splits the input into first name and last name using strings.Fields.
//
//   2. Sends a GET request to PagesJaunes (French public directory),
//      falls back to 118712.fr if nothing is found.
//
//   3. Instead of scraping CSS classes (which change constantly and break
//      scrapers), we search the raw HTML for JSON-LD structured data.
//      JSON-LD is a standard format that many directory sites embed in their
//      pages for search engines. It looks like this inside a <script> tag:
//
//        <script type="application/ld+json">
//        {
//          "@type": "Person",
//          "telephone": "+33601010101",
//          "address": {
//            "streetAddress": "7 rue du Progrès",
//            "postalCode": "75016",
//            "addressLocality": "Paris"
//          }
//        }
//        </script>
//
//      We find this block by searching for the marker string
//      `"application/ld+json"` in the raw HTML, then parse the JSON
//      between the script tags.
//
// WHY JSON-LD INSTEAD OF CSS SELECTORS?
//   CSS class names like "result-item" or "bi-bloc" change whenever a site
//   redesigns. JSON-LD schema.org Person is a W3C standard — sites embed it
//   specifically so machines can read it reliably. It's the right tool here.
//
// LIMITATIONS:
//   - Results only exist for people listed in French public directories.
//   - "Jean Dupont" is a very common French name — there may be many results
//     and we only return the first one found.
//   - Some sites block scraping even with a User-Agent. If "Not found" appears,
//     that's the site refusing the request, not a bug in the program.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// lookupFullName is called when the user runs: passive -fn "Jean Dupont"
func lookupFullName(fullName string) {
	// strings.Fields splits on any whitespace and handles multiple spaces cleanly.
	// strings.Split(s, " ") would break on double spaces — Fields is safer.
	parts := strings.Fields(fullName)
	if len(parts) < 2 {
		fmt.Println(`Error: Please provide both first and last name, e.g. "Jean Dupont"`)
		os.Exit(1)
	}

	firstName := parts[0]
	// Join the rest in case the last name has multiple words (e.g. "De La Cruz")
	lastName := strings.Join(parts[1:], " ")

	fmt.Printf("First name: %s\n", firstName)
	fmt.Printf("Last name: %s\n", lastName)

	address, phone := scrapeDirectory(firstName, lastName)

	fmt.Printf("Address: %s\n", address)
	fmt.Printf("Number: %s\n", phone)

	filename := getOutputFilename()
	content := fmt.Sprintf(
		"First name: %s\nLast name: %s\nAddress: %s\nNumber: %s\n",
		firstName, lastName, address, phone,
	)
	saveResult(content, filename)
}

// scrapeDirectory tries PagesJaunes first, then 118712.fr as a fallback.
// Returns default "Not found" strings if neither source returns a result.
func scrapeDirectory(firstName, lastName string) (address, phone string) {
	address = "Not found in public directories"
	phone = "Not found in public directories"

	addr, ph := scrapePagesJaunes(firstName, lastName)
	if addr != "" {
		return addr, ph
	}

	addr, ph = scrape118712(firstName, lastName)
	if addr != "" {
		return addr, ph
	}

	return
}

// scrapePagesJaunes queries the "Pages Blanches" (residential) section of
// PagesJaunes — the main French public phone directory.
func scrapePagesJaunes(firstName, lastName string) (address, phone string) {
	// URL-encode the name by replacing spaces with +
	query := fmt.Sprintf("%s+%s", firstName, lastName)
	url := fmt.Sprintf(
		"https://www.pagesjaunes.fr/pagesblanches/recherche?quoiqui=%s&ou=France",
		query,
	)

	resp, err := getWithAgent(url)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	return extractFromLDJSON(string(body))
}

// scrape118712 queries 118712.fr as a fallback directory source.
func scrape118712(firstName, lastName string) (address, phone string) {
	query := fmt.Sprintf("%s+%s", firstName, lastName)
	url := fmt.Sprintf(
		"https://www.118712.fr/annuaire-inverse/recherche?who=%s",
		query,
	)

	resp, err := getWithAgent(url)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	return extractFromLDJSON(string(body))
}

// extractFromLDJSON searches the raw HTML string for JSON-LD script blocks
// and tries to parse Person data from each one it finds.
//
// It loops through the page looking for every occurrence of the marker string
// `"application/ld+json"`, extracts the JSON between the opening > and
// closing </script> tag, and calls parsePersonJSON on each block.
// Returns on the first block that contains a Person with an address.
func extractFromLDJSON(content string) (address, phone string) {
	marker := `"application/ld+json"`
	start := 0

	for {
		// Find the next JSON-LD script block
		idx := strings.Index(content[start:], marker)
		if idx == -1 {
			break // No more blocks found
		}
		idx += start

		// Skip past the > that closes the opening <script> tag
		openTag := strings.Index(content[idx:], ">")
		if openTag == -1 {
			break
		}
		openTag += idx + 1

		// Find the closing </script> tag
		closeTag := strings.Index(content[openTag:], "</script>")
		if closeTag == -1 {
			break
		}

		// Extract and trim the JSON string between the tags
		jsonBlock := strings.TrimSpace(content[openTag : openTag+closeTag])

		addr, ph := parsePersonJSON(jsonBlock)
		if addr != "" {
			return addr, ph
		}

		// Move past this block and keep looking
		start = openTag + closeTag + 1
	}

	return "", ""
}

// PersonLD maps the schema.org Person JSON-LD structure.
// We only declare the fields we care about — encoding/json ignores the rest.
type PersonLD struct {
	Type      string `json:"@type"`
	Telephone string `json:"telephone"`
	Address   struct {
		StreetAddress   string `json:"streetAddress"`
		PostalCode      string `json:"postalCode"`
		AddressLocality string `json:"addressLocality"`
	} `json:"address"`
}

// GraphLD handles the case where multiple results are wrapped in a @graph array:
//
//	{ "@graph": [ { "@type": "Person", ... }, { "@type": "Person", ... } ] }
type GraphLD struct {
	Graph []PersonLD `json:"@graph"`
}

// parsePersonJSON tries to decode a JSON-LD block as a Person (or a @graph
// containing Persons) and returns the address and phone if found.
func parsePersonJSON(jsonStr string) (address, phone string) {
	// Attempt 1: direct Person object
	var person PersonLD
	if err := json.Unmarshal([]byte(jsonStr), &person); err == nil {
		if person.Type == "Person" && person.Address.StreetAddress != "" {
			address = strings.TrimSpace(fmt.Sprintf(
				"%s %s %s",
				person.Address.StreetAddress,
				person.Address.PostalCode,
				person.Address.AddressLocality,
			))
			phone = person.Telephone
			return
		}
	}

	// Attempt 2: @graph array wrapper
	var graph GraphLD
	if err := json.Unmarshal([]byte(jsonStr), &graph); err == nil {
		for _, p := range graph.Graph {
			if p.Type == "Person" && p.Address.StreetAddress != "" {
				address = strings.TrimSpace(fmt.Sprintf(
					"%s %s %s",
					p.Address.StreetAddress,
					p.Address.PostalCode,
					p.Address.AddressLocality,
				))
				phone = p.Telephone
				return
			}
		}
	}

	return "", ""
}
