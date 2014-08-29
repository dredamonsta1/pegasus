package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"
)

type runFile struct {
	Runs    []Run   `json:"runs"`
	Version float64 `json:"version"`
	Meta    struct {
		Updated time.Time `json:"updated"`
	} `json:"meta"`
}

type Importer struct {
	AccessToken string
	AppID       string
	OutputFile  string

	client *http.Client
}

func New(app_id string, access_token string, filename string) Importer {
	i := Importer{
		AccessToken: access_token,
		AppID:       app_id,
		OutputFile:  filename,
		client:      &http.Client{},
	}

	return i
}

func (i *Importer) loadFromFile() (input runFile, err error) {
	file_read, err := os.Open(i.OutputFile)
	if err != nil {
		log.Printf("Error opening output file %s: %s", i.OutputFile, err)
		return
	}

	defer file_read.Close()

	stat, err := file_read.Stat()
	if err != nil {
		log.Printf("Error stat-ing output file %s: %s", i.OutputFile, err)
		return
	}

	contents := make([]byte, stat.Size())
	_, err = file_read.Read(contents)
	if err != nil {
		log.Printf("Error reading output file %s: %s", i.OutputFile, err)
		return
	}

	err = json.Unmarshal(contents, &input)
	if err != nil {
		log.Printf("Error unmarshaling input file, starting from scratch")
		log.Printf("%s", err)
		return
	}

	log.Printf("%d runs loaded from input file %s (current as of %s)", len(input.Runs), i.OutputFile, input.Meta.Updated.Format("2006-01-02 15:04:05 -0700"))
	return input, nil
}

func (i *Importer) Import() {
	log.Println("Starting import")

	input, err := i.loadFromFile()
	runs := make([]Run, 0)

	since := time.Time{}
	until := time.Time{}
	if err == nil && !input.Meta.Updated.IsZero() {
		runs = input.Runs
		since = input.Meta.Updated.AddDate(0, 0, -1)
		until = time.Now().AddDate(0, 0, 1)
	}

	// debug
	// since = time.Now().AddDate(0, -1, 0)
	// until = time.Now()

	inc := 50
	offset := 1

	for {
		params := url.Values{
			"count":  []string{fmt.Sprintf(`%d`, inc)},
			"offset": []string{fmt.Sprintf(`%d`, offset)},
		}

		if !since.IsZero() {
			params["startDate"] = []string{since.Format("2006-01-02")}
			params["endDate"] = []string{until.Format("2006-01-02")}
		}

		body, _, err := i.request("/me/sport/activities", params)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading runs: %s", err.Error())
		}

		activities := struct {
			Data []struct {
				ActivityID   string `json:"activityId"`
				ActivityType string `json:"activityType"`
				Status       string `json:"status"`
				Device       string `json:"deviceType"`
				StartTime    string `json:"startTime"`
				Timezone     string `json:"activityTimeZone"`

				MetricSummary struct {
					Calories int64       `json:"calories,string"`
					Fuel     int64       `json:"fuel,string"`
					Distance float64     `json:"distance,string"`
					Steps    int64       `json:"steps,string"`
					Duration rawDuration `json:"duration"`
				} `json:"metricSummary"`

				Tags []struct {
					Type  string `json:"tagType"`
					Value string `json:"tagValue"`
				} `json:"tags"`
			} `json:"data"`

			Paging struct {
				Next string `json:"next"`
				Prev string `json:"previous"`
			} `json:"paging"`
		}{}
		err = json.Unmarshal(body.Bytes(), &activities)

		if len(activities.Data) == 0 {
			break
		}

		log.Printf("%d valid activities found, converting to runs (count: %d, offset: %d)", len(activities.Data), inc, offset)

		num := 0
		for _, v := range activities.Data {
			if v.ActivityID != "" && v.ActivityType == "RUN" {
				run := Run{
					ID:     v.ActivityID,
					Status: v.Status,
					Device: v.Device,

					Calories: v.MetricSummary.Calories,
					Distance: Distance(v.MetricSummary.Distance),
					Duration: time.Duration(v.MetricSummary.Duration),
					Fuel:     v.MetricSummary.Fuel,
					Steps:    v.MetricSummary.Steps,

					Tags: []Tag{},
				}

				st, err := rawTimestamp{v.StartTime, v.Timezone}.Time("America/New_York")
				if err == nil {
					run.StartTime = st
				} else {
					fmt.Println(err)
				}

				for _, t := range v.Tags {
					run.Tags = append(run.Tags, Tag{
						t.Type,
						t.Value,
					})
				}

				num += 1
				go func(run Run) {
					body, _, err := i.request("/me/sport/activities/"+run.ID+"/gps", url.Values{})

					response := struct {
						ElevationLoss  float64 `json:"elevationLoss"`
						ElevationGain  float64 `json:"elevationGain"`
						ElevationMax   float64 `json:"elevationMax"`
						ElevationMin   float64 `json:"elevationMin"`
						IntervalMetric float64 `json:"intervalMetric"`
						IntervalUnit   string  `json:"intervalUnit"`
						Waypoints      []struct {
							Latitude  float64 `json:"latitude"`
							Longitude float64 `json:"longitude"`
							Elevation float64 `json:"elevation"`
						} `json:"waypoints"`
					}{}

					if err != nil {
						num -= 1
						return
					}

					err = json.Unmarshal(body.Bytes(), &response)
					if err == nil {
						run.GPS.ElevationLoss = response.ElevationLoss
						run.GPS.ElevationGain = response.ElevationGain

						switch response.IntervalUnit {
						case "SEC":
							run.GPS.Interval = time.Duration(response.IntervalMetric) * time.Second
						}

						waypoints := []Waypoint{}
						for _, w := range response.Waypoints {
							waypoints = append(waypoints, Waypoint{
								Latitude:  w.Latitude,
								Longitude: w.Longitude,
								Elevation: w.Elevation,
							})
						}
						run.GPS.Waypoints = waypoints
					}

					runs = append(runs, run)
					num -= 1
				}(run)
			}
		}

		for num > 0 {
			time.Sleep(500 * time.Millisecond)
			// log.Printf("Waiting on %d tasks", num)
		}

		offset += inc
	}

	log.Printf("%d runs found", len(runs))

	sort.Sort(sort.Reverse(ByStartTime(runs)))

	js, err := json.Marshal(runFile{
		Runs: runs,
		Meta: struct {
			Updated time.Time `json:"updated"`
		}{
			Updated: time.Now(),
		},
		Version: 1,
	})

	file_write, err := os.OpenFile(i.OutputFile, os.O_CREATE+os.O_WRONLY+os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening %s for writing: %s", i.OutputFile, err)
	}
	defer file_write.Close()

	_, err = file_write.Write(js)
	if err != nil {
		log.Println("Error writing results to file")
	}
}

func (i *Importer) request(query string, params url.Values) (*bytes.Buffer, *http.Response, error) {

	params.Set("access_token", i.AccessToken)

	req, err := http.NewRequest("GET", "https://api.nike.com/v1"+query, nil)

	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	req.Header = http.Header{
		"appid": []string{`%appid%`},
	}
	req.Header.Add("Accept", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		log.Print("Error executing request %s: %s", query, err)
		return nil, resp, err
	}

	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)

	err_container := struct {
		Fault struct {
			Message string `json:"faultstring"`
			Details struct {
				Code string `json:"errorcode"`
			} `json:"detail"`
		} `json:"fault"`
	}{}

	err = json.Unmarshal(body.Bytes(), &err_container)
	if err == nil {
		if err_container.Fault.Details.Code == "policies.ratelimit.QuotaViolation" {
			log.Println("Request hit quota, sleeping for an hour")
			time.Sleep(60 * time.Minute)
			return i.request(query, params)
		}
	}

	return body, resp, err
}

func (this Waypoint) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`[%f, %f, %f]`, this.Latitude, this.Longitude, this.Elevation)), nil
}

func (this *Waypoint) UnmarshalJSON(in []byte) error {
	raw := []float64{}
	err := json.Unmarshal(in, &raw)
	if err != nil {
		return err
	}

	if len(raw) != 3 {
		return fmt.Errorf("Invalid waypoint: %s", string(in))
	}

	this.Latitude = raw[0]
	this.Longitude = raw[1]
	this.Elevation = raw[2]

	return nil
}
