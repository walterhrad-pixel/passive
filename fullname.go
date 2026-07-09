package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

func lookupFullName(fullName string) {
	parts := strings.Fields(fullName)
	if len(parts) < 2 {
		fmt.Println(`Error: Please provide both first and last name, e.g. "Jean Dupont"`)
		os.Exit(1)
	}

	firstName := parts[0]
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

func scrapePagesJaunes(firstName, lastName string) (address, phone string) {
	query := url.QueryEscape(firstName) + "+" + url.QueryEscape(lastName)
	reqURL := fmt.Sprintf(
		"https://www.pagesjaunes.fr/pagesblanches/recherche?quoiqui=%s&ou=France",
		query,
	)

	resp, err := getWithAgent(reqURL)
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

func scrape118712(firstName, lastName string) (address, phone string) {
	query := url.QueryEscape(firstName) + "+" + url.QueryEscape(lastName)
	reqURL := fmt.Sprintf(
		"https://www.118712.fr/annuaire-inverse/recherche?who=%s",
		query,
	)

	resp, err := getWithAgent(reqURL)
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

// Locates schema.org Person JSON-LD blocks embedded in the page HTML.
func extractFromLDJSON(content string) (address, phone string) {
	marker := `"application/ld+json"`
	start := 0

	for {
		idx := strings.Index(content[start:], marker)
		if idx == -1 {
			break
		}
		idx += start

		openTag := strings.Index(content[idx:], ">")
		if openTag == -1 {
			break
		}
		openTag += idx + 1

		closeTag := strings.Index(content[openTag:], "</script>")
		if closeTag == -1 {
			break
		}

		jsonBlock := strings.TrimSpace(content[openTag : openTag+closeTag])

		addr, ph := parsePersonJSON(jsonBlock)
		if addr != "" {
			return addr, ph
		}

		start = openTag + closeTag + 1
	}

	return "", ""
}

type PersonLD struct {
	Type      string `json:"@type"`
	Telephone string `json:"telephone"`
	Address   struct {
		StreetAddress   string `json:"streetAddress"`
		PostalCode      string `json:"postalCode"`
		AddressLocality string `json:"addressLocality"`
	} `json:"address"`
}

type GraphLD struct {
	Graph []PersonLD `json:"@graph"`
}

func parsePersonJSON(jsonStr string) (address, phone string) {
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