package importer

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
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
	body, resp, err := i.request("/me/sport/activities", url.Values{"count": []string{"1000"}})
	fmt.Println(resp, err, body)
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
