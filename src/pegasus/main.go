package main

import (
	"github.com/jimmysawczuk/go-config"

	"importer"
)

func init() {
	config.Add(config.String("api-secret", "", "Nike+ API secret", true))
	config.Build()
}

func main() {
	i := importer.New("", config.Require("api-secret").String())

	i.Import()
}
