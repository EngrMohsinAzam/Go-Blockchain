package main

import (
	"Go-Blockchain/utils"
	"fmt"
)

func main() {
	host := utils.GetHost()
	fmt.Println("Host:", host)
	// Find neighbors on ports 5000, 5001, 5002 (excluding self on 5000)
	neighbors := utils.FindNeighbors(host, 5000, 0, 0, 5000, 5002)
	fmt.Println("Neighbors (5000-5002):", neighbors)
}
