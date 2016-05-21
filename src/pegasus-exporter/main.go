package main

import (
	"github.com/jimmysawczuk/go-config"

	"importer"

	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"time"
)

func init() {
	config.Add(config.String("api-secret", "", "Nike+ API secret").Exportable(true))
	config.Add(config.String("filename", "output.json", "Filename to read/write from").Exportable(true))
	config.Add(config.String("output-dir", "out", "Directory to write converted files to"))
	config.Build()
}

type gpx struct {
	XMLName   xml.Name      `xml:"gpx"`
	Creator   string        `xml:"creator,attr"`
	Time      time.Time     `xml:"metadata>time"`
	Name      string        `xml:"trk>name"`
	Waypoints []gpxWaypoint `xml:"trk>trkseg>trkpt"`
}

type gpxWaypoint struct {
	XMLName   xml.Name  `xml:"trkpt"`
	Latitude  float64   `xml:"lat,attr"`
	Longitude float64   `xml:"lon,attr"`
	Elevation float64   `xml:"ele"`
	Time      time.Time `xml:"time"`
}

func main() {
	by, err := ioutil.ReadFile(config.Require("filename").String())
	runfile := importer.RunFile{}
	err = json.Unmarshal(by, &runfile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	os.Mkdir(config.Require("output-dir").String(), 0755)
	cnt := 0

	for _, run := range runfile.Runs {
		name := fmt.Sprintf("%s Run", run.StartTime.Format("January 2, 2006"))

		xwaypoints := make([]gpxWaypoint, len(run.GPS.Waypoints))
		t := run.StartTime

		for i, w := range run.GPS.Waypoints {
			xwaypoints[i] = gpxWaypoint{
				Latitude:  w.Latitude,
				Longitude: w.Longitude,
				Elevation: w.Elevation,
				Time:      t.Add(time.Duration(int64(i) * int64(time.Second))),
			}
		}

		xrun := gpx{
			Creator:   "pegasus",
			Time:      run.StartTime,
			Name:      name,
			Waypoints: xwaypoints,
		}

		_ = xrun

		// fmt.Println(run)

		// filename := fmt.Sprintf("run-%s.gpx", run.StartTime.In(time.UTC).Format("2006-01-02_150405"))
		// fmt.Println(filename)

		interval := time.Duration(float64(run.Duration) / float64(len(run.GPS.Waypoints)))
		if dur := math.Abs(float64(interval) - float64(time.Second)); dur > float64(time.Millisecond) && len(run.GPS.Waypoints) > 0 {
			cnt++
			// fmt.Println(run.ID, run, interval, dur)
			fmt.Println(run.StartTime, run.Duration.String(), time.Duration(int64(time.Second)*int64(len(run.GPS.Waypoints))).String())
		}

		// _ = xrun
		// by, err := xml.MarshalIndent(xrun, "", "  ")
		// fmt.Println(xml.Header+string(by), err)
		// break
		// fmt.Println("")

		// if j > 10 {
		// 	break
		// }
	}

	fmt.Println(cnt)

}
