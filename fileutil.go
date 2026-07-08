package main

import (
	"fmt"
	"os"
)

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

// 0600: output holds PII, no reason for other local users to read it
func saveResult(content, filename string) {
	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		return
	}
	fmt.Printf("Saved in %s\n", filename)
}
