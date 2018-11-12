package main

import (
	"github.com/galaco/vtf"
	"log"
)

func main() {
	_, err := vtf.ReadFromFile("samples/read/test.vtf")
	if err != nil {
		log.Fatal(err)
	}
}
