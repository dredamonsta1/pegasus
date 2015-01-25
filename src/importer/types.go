package importer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type rawDuration time.Duration
type rawTimestamp struct {
	Timestamp string
	Timezone  string
}

type Run struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"startTime"`
	Status    string    `json:"status"`
	Device    string    `json:"device"`

	Calories int64         `json:"calories"`
	Steps    int64         `json:"steps"`
	Fuel     int64         `json:"fuel"`
	Distance Distance      `json:"distance"`
	Duration time.Duration `json:"duration"`

	Tags []Tag `json:"tags"`
	GPS  GPS   `json:"gps"`
}

type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type GPS struct {
	ElevationLoss   float64       `json:"elevationLoss"`
	ElevationGain   float64       `json:"elevationGain"`
	Interval        time.Duration `json:"interval"`
	Waypoints       []Waypoint    `json:"waypoints"`
	WaypointAverage Waypoint      `json:"waypointAverage,omitempty"`
}

type Waypoint struct {
	Latitude  float64
	Longitude float64
	Elevation float64
}

type RunSlice []Run

type ByStartTime RunSlice

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
	return fmt.Sprintf(`start: %s; distance: %0.2fmi, duration: %s, pace: %s min/mi, speed: %0.2f mph, calories: %d, %d waypoints`, r.StartTime.Format("1/2/2006 3:04 PM MST"), r.Distance.Miles(), r.Duration, r.PaceMi(), r.SpeedMi(), r.Calories, len(r.GPS.Waypoints))
}

// Returns the average pace of the run in min/mi
func (r Run) PaceMi() time.Duration {
	return time.Duration(int64(float64(r.Duration) / r.Distance.Miles()))
}

// Returns the average pace of the run in min/km
func (r Run) Pace() time.Duration {
	return time.Duration(int64(float64(r.Duration) / float64(r.Distance)))
}

func (r Run) Speed() float64 {
	return float64(r.Distance) / (float64(r.Duration) / (1e9 * 60 * 60))
}

func (r Run) SpeedMi() float64 {
	return float64(r.Distance.Miles()) / (float64(r.Duration) / (1e9 * 60 * 60))
}

func (r ByStartTime) Len() int {
	return len(r)
}

func (r ByStartTime) Less(a, b int) bool {
	return r[a].StartTime.Before(r[b].StartTime)
}

func (r ByStartTime) Swap(a, b int) {
	r[a], r[b] = r[b], r[a]
}

func (g GPS) GetWaypointAverage() (sum Waypoint) {

	if cnt := len(g.Waypoints); cnt > 0 {
		for _, w := range g.Waypoints {
			sum.Latitude += w.Latitude
			sum.Longitude += w.Longitude
			sum.Elevation += w.Elevation
		}

		sum.Latitude /= float64(cnt)
		sum.Longitude /= float64(cnt)
		sum.Elevation /= float64(cnt)
	}

	return sum
}
