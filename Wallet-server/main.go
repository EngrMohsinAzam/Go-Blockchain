package main

import (
	"flag"
	"log"
)

func init() {
	log.SetPrefix("Wallet Server: ")
}
func main() {
	port := flag.Uint("port", 8080, "TCP Port for Wallet Server")
	gateway := flag.String("gateway", "http://localhost:5000", "Blockchain Server Gateway URL")
	flag.Parse()

	app := NewWalletServer(uint16(*port), *gateway)
	app.Run()
}
