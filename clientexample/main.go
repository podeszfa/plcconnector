package main

import (
	"fmt"
	"os"

	plc ".."
)

func main() {
	addr := "localhost:44818"
	if len(os.Args) > 1 {
		if os.Args[1] == "d" {
			ids, err := plc.Discover()
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, id := range ids {
				fmt.Printf("%#v\n", id)
			}
		} else {
			addr = os.Args[1]
		}
	}
	_, err := plc.Connect(addr)
	if err != nil {
		fmt.Println(err)
		return
	}
}