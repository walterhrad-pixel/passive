package main

// main.go
//
// This is the entry point of the program — the first thing Go runs.
//
// WHAT THIS FILE DOES:
//   1. Defines the three CLI flags: -fn, -ip, -u
//   2. Parses the arguments the user typed
//   3. Calls the right function based on which flag was used
//
// It deliberately contains NO business logic — it only reads flags and
// delegates to the functions in ip.go, username.go, and fullname.go.
// This is a clean separation of concerns: main.go knows WHAT to do,
// the other files know HOW to do it.
//
// FLAG PARSING IN GO:
//   Go's built-in "flag" package handles CLI arguments.
//   flag.String("fn", "", "description") registers a -fn flag that holds
//   a string value, defaulting to "". After flag.Parse() is called,
//   the pointer *fn holds whatever the user typed.
//
// USAGE EXAMPLES:
//   passive --help
//   passive -fn "Jean Dupont"
//   passive -ip 127.0.0.1
//   passive -u "@user01"

import (
	"flag"
	"fmt"
	"os"
)

// printHelp prints the help text in the exact format the project specifies.
// We assign this to flag.Usage so it's also shown when the user runs --help.
func printHelp() {
	fmt.Println("Welcome to passive v1.0.0")
	fmt.Println("\nOPTIONS:")
	fmt.Println("    -fn         Search with full-name")
	fmt.Println("    -ip         Search with ip address")
	fmt.Println("    -u          Search with username")
}

func main() {
	// Register the three flags.
	// flag.String returns a *string — a pointer to where the value will be stored.
	// We dereference with *fn, *ip, *u when reading the values below.
	fn := flag.String("fn", "", "Search with full-name")
	ip := flag.String("ip", "", "Search with ip address")
	u  := flag.String("u",  "", "Search with username")

	// Replace the default usage message with our custom printHelp function.
	// This is what gets printed when the user runs `passive --help`.
	flag.Usage = printHelp
	flag.Parse()

	// If no arguments were passed, print help and exit cleanly.
	// len(os.Args) is 1 when the program is run with no arguments
	// (os.Args[0] is always the program name itself).
	if len(os.Args) == 1 {
		printHelp()
		os.Exit(0)
	}

	// Route to the right function based on which flag was provided.
	// A switch with no condition works like a chain of if-else — it runs
	// the first case whose condition is true.
	switch {
	case *fn != "":
		lookupFullName(*fn)   // defined in fullname.go
	case *ip != "":
		lookupIP(*ip)         // defined in ip.go
	case *u != "":
		checkUsername(*u)     // defined in username.go
	default:
		// A flag was passed but none of the three we know about.
		printHelp()
		os.Exit(1)
	}
}
