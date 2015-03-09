package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jimmysawczuk/go-config"

	"fmt"
	"importer"
	"log"
	"strconv"
)

func init() {
	config.Add(config.String("api-secret", "", "Nike+ API secret", true))
	config.Add(config.String("filename", "output.json", "Filename to read/write from", true))
	config.Add(config.String("address", ":3000", "Listen on this address", true))
	config.Build()
}

type WebRun struct {
	importer.Run
	NECorner Coord
	SWCorner Coord
}

var runs []WebRun

type Coord struct {
	Lat, Lng float64
}

type Bounds struct {
	NE Coord
	SW Coord
}

func main() {
	file_ch := make(chan bool)
	go func() {
		i := importer.New("", "", config.Require("filename").String())

		rf, _ := i.LoadFromFile()
		runs = make([]WebRun, 0)
		for _, run := range rf.Runs {
			wr := WebRun{Run: run}
			wr.NECorner, wr.SWCorner = wr.GetBounds()
			runs = append(runs, wr)

		}

		file_ch <- true
	}()

	r := gin.Default()
	r.LoadHTMLGlob("src/pegasus-web-templates/*")

	r.GET("", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{})
	})

	r.GET("/api/v1/runs", func(c *gin.Context) {
		log.Printf("Serving %d runs", len(runs))
		// c.JSON(200, importer.WithoutGPS(runs))
		c.JSON(200, nil)
		log.Printf("Done")
	})

	r.GET("/api/v1/heatmap", func(c *gin.Context) {
		c.Request.ParseForm()

		bounds := Bounds{}
		bounds.NE.Lat, _ = strconv.ParseFloat(c.Request.Form.Get("bounds[ne][lat]"), 64)
		bounds.NE.Lng, _ = strconv.ParseFloat(c.Request.Form.Get("bounds[ne][lng]"), 64)
		bounds.SW.Lat, _ = strconv.ParseFloat(c.Request.Form.Get("bounds[sw][lat]"), 64)
		bounds.SW.Lng, _ = strconv.ParseFloat(c.Request.Form.Get("bounds[sw][lng]"), 64)

		type ptCnt struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
			Cnt int64   `json:"cnt"`
		}

		pt_map := make(map[string]*ptCnt)

		for _, run := range runs {
			if run.NECorner.Lat < bounds.NE.Lat && run.SWCorner.Lat > bounds.SW.Lat &&
				run.NECorner.Lng < bounds.NE.Lng && run.SWCorner.Lng > bounds.SW.Lng {
				for _, pt := range run.GPS.Waypoints {
					coord := Coord{
						pt.Latitude,
						pt.Longitude,
					}

					coord.Round(5e-4)
					hash := coord.Hash()

					if _, exists := pt_map[hash]; !exists {
						pt_map[hash] = &ptCnt{
							Lat: coord.Lat,
							Lng: coord.Lng,
						}
					}

					pt_map[hash].Cnt++
				}
			}
		}

		pt_arr := make([]ptCnt, 0)
		for _, cnt := range pt_map {
			pt_arr = append(pt_arr, *cnt)
		}

		c.JSON(200, pt_arr)
	})

	// _ = <-file_ch

	r.Run(config.Require("address").String())
}

func (c *Coord) Round(precision float64) {
	mult := 1 / precision
	c.Lat *= mult
	c.Lng *= mult

	c.Lat = float64(int64(c.Lat)) / mult
	c.Lng = float64(int64(c.Lng)) / mult
}

func (c Coord) Hash() string {
	return fmt.Sprintf("%v,%v", c.Lat, c.Lng)
}

func (w WebRun) GetBounds() (ne Coord, sw Coord) {
	if len(w.GPS.Waypoints) == 0 {
		return
	}

	ne = Coord{
		Lat: w.GPS.Waypoints[0].Latitude,
		Lng: w.GPS.Waypoints[0].Longitude,
	}
	sw = ne

	for _, wp := range w.GPS.Waypoints {
		if wp.Latitude > ne.Lat {
			ne.Lat = wp.Latitude
		}

		if wp.Latitude < sw.Lat {
			sw.Lat = wp.Latitude
		}

		if wp.Longitude > ne.Lng {
			ne.Lng = wp.Longitude
		}

		if wp.Longitude < sw.Lat {
			sw.Lng = wp.Longitude
		}
	}

	return
}
