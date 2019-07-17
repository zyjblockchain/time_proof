package main

import (
	"fmt"
	"time_proof/client"
)

func main() {
	if err := client.Client(10); err != nil {
		fmt.Printf("error: %v\n", err)
	}

	// fmt.Println(time.Now().Unix())
	// fmt.Println(time.Now().UnixNano())
	// fmt.Println(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	//
	// var temp int64
	// temp = 1246812584719553223
	// temp = temp * 10
	// if temp > math.MaxInt64 /2{
	// 	log.Println("aaaaaa",temp)
	// } else {
	// 	log.Println(temp)
	// }

}
