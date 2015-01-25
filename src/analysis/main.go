package main

import (
	"github.com/jimmysawczuk/go-config"

	"importer"

	"fmt"
)

func init() {
	config.Add(config.String("filename", "output.json", "Filename to read/write from", true))
	config.Build()
}

func main() {
	i := importer.New("", "", config.Require("filename").String())

	rf, _ := i.LoadFromFile()
	for _, run := range rf.Runs[0:30] {
		avg := run.GPS.GetWaypointAverage()
		fmt.Printf("%s: %0.3f %0.3f %0.3f\n", run.StartTime, avg.Latitude, avg.Longitude, avg.Elevation)
	}
}
