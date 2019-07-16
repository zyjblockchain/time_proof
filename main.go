package main

import (
	"fmt"
	"time_proof/client"
)

func main() {
	if err := client.Client(5); err != nil {
		fmt.Printf("error: %v", err)
	}
}
