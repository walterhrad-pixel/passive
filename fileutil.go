package main

// fileutil.go
//
// This file handles everything related to saving results to disk.
//
// The project requirement says:
//   - Results must be saved to result.txt
//   - If result.txt already exists, save to result2.txt
//   - If result2.txt exists too, save to result3.txt, and so on
//
// Both functions here are used by ip.go, username.go, and fullname.go —
// they call getOutputFilename() to find the right file, then saveResult()
// to write it.

import (
	"fmt"
	"os"
)

// getOutputFilename returns the next available result filename.
//
// It starts with "result.txt" and checks if the file exists using os.Stat().
// os.Stat() returns an error if the file doesn't exist — we check specifically
// for os.IsNotExist(err) to confirm it's a "file not found" error (not a
// permissions error or something else).
//
// Once it finds a name that doesn't exist yet, it returns that name.
//
// Example flow:
//
//	result.txt exists   → try result2.txt
//	result2.txt exists  → try result3.txt
//	result3.txt missing → return "result3.txt"
func getOutputFilename() string {
	name := "result.txt"
	counter := 2
	for {
		if _, err := os.Stat(name); os.IsNotExist(err) {
			return name
		}
		name = fmt.Sprintf("result%d.txt", counter)
		counter++
	}
}

// saveResult writes content to the given filename and prints confirmation.
//
// os.WriteFile creates the file if it doesn't exist, or overwrites it if it does.
// The 0644 is a Unix file permission: owner can read/write, everyone else can read.
// This is the standard permission for normal text files.
func saveResult(content, filename string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		return
	}
	fmt.Printf("Saved in %s\n", filename)
}
