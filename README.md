# passive

A command-line tool for passive reconnaissance. You give it a name, an IP address, or a username — it goes and finds what it can from public sources.

I built this as part of my cybersecurity coursework at Zone01 Kisumu. The idea behind passive recon is simple: before you ever touch a target, there's already a surprising amount of information about it sitting in public directories, databases, and social platforms. This tool pulls some of that together.

---

## What it does

**Full name search (`-fn`)**
Give it a name and it searches French public phone directories for an address and phone number. It tries PagesJaunes first, then falls back to 118712.fr if nothing comes up.

**IP address lookup (`-ip`)**
Give it an IP and it returns the ISP and geographic coordinates. Uses ip-api.com under the hood — no API key needed. If you pass a private IP like 127.0.0.1, it won't crash, it'll just tell you that's a loopback address with no ISP.

**Username search (`-u`)**
Give it a username and it checks 7 platforms to see if an account exists there. It does this by making an HTTP request to each profile URL and reading the response code — 200 means the profile loaded, 404 or a redirect usually means it doesn't exist.

Every time you run a command the result gets saved to a file. First run saves to `result.txt`, second run saves to `result2.txt`, and so on — it never overwrites.

---

## Installation

You'll need Go installed. If you don't have it yet, grab it from [golang.org](https://golang.org/dl/).

```bash
git clone https://github.com/yourusername/passive.git
cd passive
go build -o passive .
```

If you want to run it from anywhere without typing the full path:

```bash
sudo mv passive /usr/local/bin/
```

---

## Usage

```bash
passive --help
```

```
Welcome to passive v1.0.0

OPTIONS:
    -fn         Search with full-name
    -ip         Search with ip address
    -u          Search with username
```

### Search by full name

```bash
passive -fn "Jean Dupont"
```

```
First name: Jean
Last name: Dupont
Address: 7 rue du Progrès 75016 Paris
Number: +33601010101
Saved in result.txt
```

### Search by IP address

```bash
passive -ip 8.8.8.8
```

```
ISP: Google LLC
City Lat/Lon:   (37.4056) / (-122.0775)
Saved in result2.txt
```

### Search by username

```bash
passive -u "@torvalds"
```

```
Facebook : no
Twitter : yes
Instagram : no
GitHub : yes
LinkedIn : no
TikTok : no
Reddit : yes
Saved in result3.txt
```

---

## How it actually works

### The IP lookup

This one is the most straightforward. The tool sends a GET request to `http://ip-api.com/json/<ip>` and decodes the JSON response. ip-api is free and doesn't require registration. For private IP ranges (127.x.x.x, 192.168.x.x, 10.x.x.x etc.) the tool skips the network request entirely and tells you what that range means instead of failing.

### The username check

For each platform, the tool builds the standard profile URL and fires off a request. The response status code is what tells us if the account exists:

- `200` — page loaded, account likely exists
- `404` — page not found, account doesn't exist
- `301/302` — redirect, which usually means the platform is sending you to a login or error page (i.e. the account doesn't exist)
- `403/429` — the platform blocked the request entirely

One thing worth knowing: Facebook and LinkedIn aggressively block automated requests. If they show up as `blocked` even for real accounts, that's expected — those platforms make passive recon harder by design.

### The full name lookup

This one uses web scraping. The tool sends a request to PagesJaunes with the name as a search query. Instead of trying to parse CSS class names (which break constantly whenever a site redesigns), it looks for JSON-LD structured data embedded in the HTML. JSON-LD is a standard format that directory sites include for search engines — it looks like this somewhere in the page source:

```json
{
  "@type": "Person",
  "telephone": "+33601010101",
  "address": {
    "streetAddress": "7 rue du Progrès",
    "postalCode": "75016",
    "addressLocality": "Paris"
  }
}
```

The tool finds that block, parses it, and pulls out the address and phone. If the site doesn't return anything — common name with no listing, or the site blocked the request — it returns "Not found in public directories" rather than crashing.

---

## Project structure

```
passive/
├── main.go         entry point, reads flags and routes to the right function
├── httpclient.go   shared HTTP client used by everything else
├── ip.go           IP lookup and private IP classification
├── username.go     username existence checks across platforms
├── fullname.go     directory scraping and JSON-LD parsing
├── fileutil.go     result file naming and writing
├── go.mod          Go module definition
└── README.md
```

The code is split this way so each file has one clear job. If something breaks in the username checker you know exactly where to look, same for the scraper or the IP logic.

---

## Limitations worth knowing

- Full name results only exist for people listed in French public directories. This tool was built in a French-curriculum context — for other countries you'd need different directory sources.
- Platform checks are based on HTTP status codes, not page content. Some platforms return 200 for non-existent users and show an error inside the HTML body — catching that would require deeper parsing.
- This tool makes no attempt to bypass bot protection. If a platform blocks the request, that's the result you get.

---

## Dependencies

None beyond the Go standard library. No external packages, no API keys, no accounts needed.

---

## Legal note

This tool only uses publicly available information and makes no attempt to bypass authentication or access anything private. That said — passive recon on someone without their knowledge can still cross legal and ethical lines depending on your jurisdiction and intent. Only use this on yourself or with explicit permission from whoever you're researching. It was built for learning purposes.
