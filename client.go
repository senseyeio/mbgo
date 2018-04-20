package mbgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/senseyeio/mbgo/internal/rest"
)

type Client struct {
	cli *rest.Client
}

func NewClient(cli *http.Client, root *url.URL) *Client {
	return &Client{
		cli: rest.NewClient(cli, root),
	}
}

type Imposter struct {
	Port         int    `json:"port"`
	Proto        string `json:"protocol"`
	Name         string `json:"name"`
	RequestCount int    `json:"numberOfRequests,omitempty"`
}

func (cli *Client) Create(imp Imposter) (*Imposter, error) {
	p := "/imposters"
	b, err := json.Marshal(&imp)
	if err != nil {
		return nil, err
	}

	req, err := cli.cli.BuildRequest(http.MethodPost, p, bytes.NewReader(b), nil)
	if err != nil {
		return nil, err
	}

	if err := cli.cli.ProcessRequest(req, http.StatusCreated, &imp); err != nil {
		return nil, err
	}
	return &imp, nil
}

func (cli *Client) Imposter(port int, replay bool) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d", port)
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.cli.BuildRequest(http.MethodGet, p, nil, nil)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &imp); err != nil {
		return nil, err
	}
	return &imp, nil
}

func (cli *Client) Delete(port int, replay bool) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d", port)
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.cli.BuildRequest(http.MethodDelete, p, nil, vs)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &imp); err != nil {
		return nil, err
	}
	return &imp, nil
}

func (cli *Client) DeleteRequests(port int) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d/requests", port)

	req, err := cli.cli.BuildRequest(http.MethodDelete, p, nil, nil)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &imp); err != nil {
		return nil, err
	}
	return &imp, nil
}

type imposterListWrapper struct {
	Imposters []Imposter `json:"imposters"`
}

func (cli *Client) Overwrite(imps []Imposter, replay bool) ([]Imposter, error) {
	p := "/imposters"
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))
	b, err := json.Marshal(&struct {
		Imposters []Imposter `json:"imposters"`
	}{
		Imposters: imps,
	})
	if err != nil {
		return nil, err
	}

	req, err := cli.cli.BuildRequest(http.MethodPut, p, bytes.NewReader(b), vs)
	if err != nil {
		return nil, err
	}

	var wrap imposterListWrapper
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &wrap); err != nil {
		return nil, err
	}
	return wrap.Imposters, nil
}

func (cli *Client) Imposters(replay bool) ([]Imposter, error) {
	p := "/imposters"
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.cli.BuildRequest(http.MethodGet, p, nil, vs)
	if err != nil {
		return nil, err
	}

	var wrap imposterListWrapper
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &wrap); err != nil {
		return nil, err
	}
	return wrap.Imposters, nil
}

func (cli *Client) DeleteAll(replay bool) ([]Imposter, error) {
	p := "/imposters"
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.cli.BuildRequest(http.MethodDelete, p, nil, vs)
	if err != nil {
		return nil, err
	}

	var wrap imposterListWrapper
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &wrap); err != nil {
		return nil, err
	}
	return wrap.Imposters, nil
}

type Config struct {
	Version string `json:"version"`
	Options struct {
		Help           bool     `json:"help"`
		NoParse        bool     `json:"noParse"`
		NoLogFile      bool     `json:"nologfile"`
		AllowInjection bool     `json:"allowInjection"`
		LocalOnly      bool     `json:"localOnly"`
		Mock           bool     `json:"mock"`
		Debug          bool     `json:"debug"`
		Port           int      `json:"port"`
		PIDFile        string   `json:"pidfile"`
		LogFile        string   `json:"logfile"`
		LogLevel       string   `json:"loglevel"`
		IPWhitelist    []string `json:"ipWhitelist"`
	} `json:"options"`
	Process struct {
		NodeVersion  string  `json:"nodeVersion"`
		Architecture string  `json:"architecture"`
		Platform     string  `json:"platform"`
		RSS          int64   `json:"rss"`
		HeapTotal    int64   `json:"heapTotal"`
		HeapUsed     int64   `json:"heapUsed"`
		Uptime       float64 `json:"uptime"`
		CWD          string  `json:"cwd"`
	} `json:"process"`
}

func (cli *Client) Config() (*Config, error) {
	p := "/config"

	req, err := cli.cli.BuildRequest(http.MethodGet, p, nil, nil)
	if err != nil {
		return nil, err
	}

	cfg := Config{}
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Log struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type logListWrapper struct {
	Logs []Log `json:"logs"`
}

func (cli *Client) Logs(start, end int) ([]Log, error) {
	p := "/logs"
	vs := url.Values{}
	if start >= 0 {
		vs.Add("startIndex", strconv.Itoa(start))
	}
	if end >= 0 {
		vs.Add("endIndex", strconv.Itoa(end))
	}

	req, err := cli.cli.BuildRequest(http.MethodGet, p, nil, vs)
	if err != nil {
		return nil, err
	}

	var wrap logListWrapper
	if err := cli.cli.ProcessRequest(req, http.StatusOK, &wrap); err != nil {
		return nil, err
	}
	return wrap.Logs, nil
}
