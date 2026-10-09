package main

import (
	"github.com/galaco/vtf"
	"log"
)

func main() {
	_, err := vtf.ReadFromFile("testdata/test.vtf")
	if err != nil {
		log.Fatal(err)
	}
}
