package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jimmysawczuk/go-config"

	"importer"
	"log"
)

func init() {
	config.Add(config.String("api-secret", "", "Nike+ API secret", true))
	config.Add(config.String("filename", "output.json", "Filename to read/write from", true))
	config.Add(config.String("address", ":3000", "Listen on this address", true))
	config.Build()
}

var runs []importer.Run

func main() {
	file_ch := make(chan bool)
	go func() {
		i := importer.New("", "", config.Require("filename").String())

		rf, _ := i.LoadFromFile()
		runs = rf.Runs

		log.Println("Runs file loaded")
		file_ch <- true
	}()

	r := gin.Default()

	r.GET("", func(c *gin.Context) {
		c.String(200, "yay!")
	})

	r.GET("/api/v1/runs", func(c *gin.Context) {
		c.JSON(200, runs)
	})

	_ = <-file_ch

	r.Run(config.Require("address").String())
}
