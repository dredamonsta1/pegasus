package main

import (
	"importer"

	// "fmt"
)

func main() {
	i := importer.New("", "5e366cacfa97c9a0abd76ca5ece78aeb")

	i.Import()
}
