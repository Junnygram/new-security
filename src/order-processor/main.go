package main

import (
	"log"
	"time"
)

func main() {
	log.Println("Order Processor started...")
	for {
		log.Println("Checking for new orders...")
		// Simulate processing work
		time.Sleep(10 * time.Second)
	}
}
