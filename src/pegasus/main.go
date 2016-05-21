package main

import (
	"github.com/jimmysawczuk/go-config"

	"importer"
)

func init() {
	config.Add(config.String("api-secret", "", "Nike+ API secret").Exportable(true))
	config.Add(config.String("filename", "output.json", "Filename to read/write from").Exportable(true))
	config.Build()
}

func main() {
	i := importer.New("", config.Require("api-secret").String(), config.Require("filename").String())

	i.Import()
}
