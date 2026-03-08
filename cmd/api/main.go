package main

import (
	"go-timekeeper/internal/kernel"
	"log"
)

// main starts the application kernel.
func main() {
	k, err := kernel.New()
	if err != nil {
		log.Fatalf("Failed to create kernel: %v", err)
	}

	if err := k.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
