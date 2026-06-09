package main

import (
	"flag"
	"log"
)

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {
	port := flag.Uint("port", 5000, "TCP Port for Blockchain Server")
	flag.Parse()

	// Create blockchain with miner address
	minerAddress := "miner_blockchain_address_123"
	blockchain := NewBlockchain(minerAddress)

	log.Printf("Blockchain initialized with miner address: %s", minerAddress)

	// Create and run blockchain server
	app := NewBlockchainServer(uint16(*port), blockchain)
	app.Run()
}
