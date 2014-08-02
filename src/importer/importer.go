package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Importer struct {
	AccessToken string
	AppID       string

	client *http.Client
}

func New(app_id string, access_token string) Importer {
	i := Importer{
		AccessToken: access_token,
		AppID:       app_id,
		client:      &http.Client{},
	}

	return i
}

func (i *Importer) Import() {
	log.Println("Starting import")

	runs := make([]Run, 0)

	inc := 50
	offset := 1

	for {
		body, _, err := i.request("/me/sport/activities", url.Values{"count": []string{fmt.Sprintf(`%d`, inc)}, "offset": []string{fmt.Sprintf(`%d`, offset)}})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading runs: %s", err.Error())
		}

		activities := struct {
			Data []struct {
				ActivityID   string `json:"activityId"`
				ActivityType string `json:"activityType"`
				Status       string `json:"status"`
				Device       string `json:"device"`
				StartTime    string `json:"startTime"`
				Timezone     string `json:"activityTimeZone"`

				MetricSummary struct {
					Calories int64       `json:"calories"`
					Fuel     int64       `json:"fuel"`
					Distance float64     `json:"distance"`
					Steps    int64       `json:"steps"`
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

		// for _, run := range runs {
		// 	log.Println(run.String())
		// }

		offset += inc
	}

	log.Printf("%d runs saved", len(runs))

	js, err := json.Marshal(runs)
	if err == nil {
		fmt.Println(string(js))
	} else {
		fmt.Println(err)
	}
}

func (i *Importer) request(query string, params url.Values) (*bytes.Buffer, *http.Response, error) {

	params.Set("access_token", i.AccessToken)

	req, err := http.NewRequest("GET", "https://api.nike.com"+query, nil)

	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	req.Header = http.Header{
		"appid": []string{`%appid%`},
	}
	req.Header.Add("Accept", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, resp, err
	}

	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)

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
