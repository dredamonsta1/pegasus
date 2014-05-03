package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Importer struct {
	AccessToken string
	AppID       string

	client *http.Client
}

type rawDuration time.Duration
type rawTimestamp struct {
	Timestamp string
	Timezone  string
}
type rawTag struct {
	Type  string `json:"tagType"`
	Value string `json:"tagValue"`
}

type Tag struct {
	Name  string
	Value string
}

type rawMetricSummary struct {
	Calories int64       `json:"calories"`
	Fuel     int64       `json:"fuel"`
	Distance float64     `json:"distance"`
	Steps    int64       `json:"steps"`
	Duration rawDuration `json:"duration"`
}

type rawActivity struct {
	ActivityID   string `json:"activityId"`
	ActivityType string `json:"activityType"`
	Status       string `json:"status"`
	Device       string `json:"device"`
	StartTime    string `json:"startTime"`
	Timezone     string `json:"activityTimeZone"`

	MetricSummary rawMetricSummary `json:"metricSummary"`

	Tags []rawTag `json:"tags"`
}

type aggregateActivities struct {
	Data []rawActivity `json:"data"`
}

type Run struct {
	ID        string
	StartTime time.Time
	Status    string
	Device    string

	Calories int64
	Steps    int64
	Fuel     int64
	Distance Distance
	Duration time.Duration

	Tags []Tag
}

type Distance float64

func (d Distance) Kilometers() float64 {
	return float64(d)
}

func (d Distance) Miles() float64 {
	return float64(d) / 1.60934
}

func (t rawTimestamp) Time(tz string) (time.Time, error) {
	in_tz, err := time.LoadLocation(t.Timezone)
	if err != nil {
		return time.Time{}, err
	}

	out_tz, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, err
	}

	v, err := time.Parse(`2006-01-02T15:04:05Z`, t.Timestamp)
	if err != nil {
		return time.Time{}, err
	}

	v = v.In(in_tz)

	u := time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), 0, out_tz)

	return u, nil
}

func (t *rawDuration) UnmarshalJSON(in []byte) (err error) {
	var str string
	err = json.Unmarshal(in, &str)
	if err != nil {
		return err
	}

	frac := strings.Split(str, ".")
	whole := strings.Split(frac[0], ":")

	if len(frac) == 1 {
		frac[1] = "0"
	}

	if len(whole) != 3 {
		return fmt.Errorf("Invalid duration")
	}

	v, err := time.ParseDuration(fmt.Sprintf(`%sh%sm%ss.%sms`, whole[0], whole[1], whole[2], frac[1]))
	if err != nil {
		return err
	}

	*t = rawDuration(v)
	return nil
}

func (r Run) String() string {
	return fmt.Sprintf(`start: %s; distance: %0.2fmi, duration: %s, calories: %d`, r.StartTime.Format("1/2/2006 3:04 PM MST"), r.Distance.Miles(), r.Duration, r.Calories)
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
	body, _, err := i.request("/me/sport/activities", url.Values{"count": []string{"50"}})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading runs: %s", err.Error())
	}

	out := bytes.Buffer{}
	json.Indent(&out, body.Bytes(), "", "  ")
	fmt.Println(out.String())

	activities := aggregateActivities{}
	err = json.Unmarshal(body.Bytes(), &activities)

	log.Printf("%d valid activities found, converting to runs", len(activities.Data))

	runs := make([]Run, 0)
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

			runs = append(runs, run)
		}
	}

	for _, v := range runs {
		fmt.Println(v.String())
	}

	activities = aggregateActivities{}
}

func (i *Importer) request(query string, params url.Values) (body *bytes.Buffer, resp *http.Response, err error) {

	params.Set("access_token", i.AccessToken)

	req, err := http.NewRequest("GET", "https://api.nike.com"+query, nil)

	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	req.Header = http.Header{
		"appid": []string{`%appid%`},
	}
	req.Header.Add("Accept", "application/json")

	resp, err = i.client.Do(req)

	body = &bytes.Buffer{}
	body.ReadFrom(resp.Body)

	return
}
