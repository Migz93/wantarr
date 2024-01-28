package pvr

import (
	"fmt"
	"strings"
	"time"

	"github.com/imroc/req"
	"github.com/migz93/wantarr/config"
	"github.com/migz93/wantarr/logger"
	"github.com/migz93/wantarr/utils/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/* Structs */

type WhisparrV2 struct {
	cfg        *config.Pvr
	log        *logrus.Entry
	apiUrl     string
	reqHeaders req.Header
	timeout    int
}

type WhisparrV2Queue struct {
	Size int `json:"totalRecords"`
}

type WhisparrV2Episode struct {
	Id         int
	AirDateUtc string `json:"releaseDate"`
	Monitored  bool
}

type WhisparrV2Wanted struct {
	Page          int
	PageSize      int
	SortKey       string
	SortDirection string
	TotalRecords  int
	Records       []WhisparrV2Episode
}

type WhisparrV2SystemStatus struct {
	Version string
}

type WhisparrV2CommandStatus struct {
	Name    string
	Message string
	Started time.Time
	Ended   time.Time
	Status  string
}

type WhisparrV2CommandResponse struct {
	Id int
}

type WhisparrV2EpisodeSearch struct {
	Name     string `json:"name"`
	Episodes []int  `json:"episodeIds"`
}

/* Initializer */

func NewWhisparrV2(name string, c *config.Pvr) *WhisparrV2 {
	// set api url
	apiUrl := ""
	if strings.Contains(c.URL, "/api") {
		apiUrl = c.URL
	} else {
		apiUrl = web.JoinURL(c.URL, "/api/v3")
	}

	// set headers
	reqHeaders := req.Header{
		"X-Api-Key": c.ApiKey,
	}

	return &WhisparrV2{
		cfg:        c,
		log:        logger.GetLogger(name),
		apiUrl:     apiUrl,
		reqHeaders: reqHeaders,
		timeout:    pvrDefaultTimeout,
	}
}

/* Private */

func (p *WhisparrV2) getSystemStatus() (*WhisparrV2SystemStatus, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/system/status"), p.timeout, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving system status api response from whisparr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid system status api response from whisparr: %s",
			resp.Response().Status)
	}

	// decode response
	var s WhisparrV2SystemStatus
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding system status api response from whisparr")
	}

	return &s, nil
}

func (p *WhisparrV2) getCommandStatus(id int) (*WhisparrV2CommandStatus, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, fmt.Sprintf("/command/%d", id)), p.timeout,
		p.reqHeaders, &pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving command status api response from whisparr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid command status api response from whisparr: %s",
			resp.Response().Status)
	}

	// decode response
	var s WhisparrV2CommandStatus
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding command status api response from whisparr")
	}

	return &s, nil
}

/* Interface Implements */

func (p *WhisparrV2) Init() error {
	// retrieve system status
	status, err := p.getSystemStatus()
	if err != nil {
		return errors.Wrap(err, "failed initializing whisparr pvr")
	}

	// determine version
	switch status.Version[0:1] {
	case "2":
		break
	default:
		return fmt.Errorf("unsupported version of whisparr pvr: %s", status.Version)
	}
	return nil
}

func (p *WhisparrV2) GetQueueSize() (int, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/queue"), p.timeout, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return 0, errors.WithMessage(err, "failed retrieving queue api response from whisparr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return 0, fmt.Errorf("failed retrieving valid queue api response from whisparr: %s",
			resp.Response().Status)
	}

	// decode response
	var q WhisparrV2Queue
	if err := resp.ToJSON(&q); err != nil {
		return 0, errors.WithMessage(err, "failed decoding queue api response from whisparr")
	}

	p.log.WithField("queue_size", q.Size).Debug("Queue retrieved")
	return q.Size, nil
}

func (p *WhisparrV2) GetWantedMissing() ([]MediaItem, error) {
	// logic vars
	totalRecords := 0
	var wantedMissing []MediaItem

	page := 1
	lastPageSize := pvrDefaultPageSize

	// set params
	params := req.QueryParam{
		"sortKey":   "airDateUtc",
		"pageSize":  pvrDefaultPageSize,
		"monitored": "true",
	}

	// retrieve all page results
	p.log.Info("Retrieving wanted missing media...")

	for {
		// break loop when all pages retrieved
		if lastPageSize < pvrDefaultPageSize {
			break
		}

		// set page
		params["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/wanted/missing"), p.timeout,
			p.reqHeaders, &pvrDefaultRetry, params)
		if err != nil {
			return nil, errors.WithMessage(err, "failed retrieving wanted missing api response from whisparr")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid wanted missing api response from whisparr: %s",
				resp.Response().Status)
		}

		// decode response
		var m WhisparrV2Wanted
		if err := resp.ToJSON(&m); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding wanted missing api response from whisparr")
		}

		// process response
		lastPageSize = len(m.Records)
		for _, episode := range m.Records {

			// store this episode
			//extraDate := "T00:00:00Z"
			//airDate := episode.AirDateUtc

			airDate, _ := time.Parse("2006-01-02", episode.AirDateUtc)
			//airDateNew := airDate+extraDate
			wantedMissing = append(wantedMissing, MediaItem{
				ItemId:     episode.Id,
				AirDateUtc: airDate,
				LastSearch: time.Time{},
			})
		}
		totalRecords += lastPageSize

		p.log.WithField("page", page).Debug("Retrieved")
		page += 1

		// close response
		_ = resp.Response().Body.Close()
	}

	p.log.WithField("media_items", totalRecords).Info("Finished")

	return wantedMissing, nil
}

func (p *WhisparrV2) GetWantedCutoff() ([]MediaItem, error) {
	// logic vars
	totalRecords := 0
	var wantedCutoff []MediaItem

	page := 1
	lastPageSize := pvrDefaultPageSize

	// set params
	params := req.QueryParam{
		"sortKey":   "airDateUtc",
		"pageSize":  pvrDefaultPageSize,
		"monitored": "true",
	}

	// retrieve all page results
	p.log.Info("Retrieving wanted cutoff unmet media...")

	for {
		// break loop when all pages retrieved
		if lastPageSize < pvrDefaultPageSize {
			break
		}

		// set page
		params["page"] = page

		// send request
		resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/wanted/cutoff"), p.timeout,
			p.reqHeaders, &pvrDefaultRetry, params)
		if err != nil {
			return nil, errors.WithMessage(err, "failed retrieving wanted cutotff unmet api response from whisparr")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid wanted cutoff unmet api response from whisparr: %s",
				resp.Response().Status)
		}

		// decode response
		var m WhisparrV2Wanted
		if err := resp.ToJSON(&m); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding wanted cutoff unmet api response from whisparr")
		}

		// process response
		lastPageSize = len(m.Records)
		for _, episode := range m.Records {
			// store this episode
			airDate, _ := time.Parse("2006-01-02", episode.AirDateUtc)
			wantedCutoff = append(wantedCutoff, MediaItem{
				ItemId:     episode.Id,
				AirDateUtc: airDate,
				LastSearch: time.Time{},
			})
		}
		totalRecords += lastPageSize

		p.log.WithField("page", page).Debug("Retrieved")
		page += 1

		// close response
		_ = resp.Response().Body.Close()
	}

	p.log.WithField("media_items", totalRecords).Info("Finished")

	return wantedCutoff, nil
}

func (p *WhisparrV2) SearchMediaItems(mediaItemIds []int) (bool, error) {
	// set request data
	payload := WhisparrV2EpisodeSearch{
		Name:     "EpisodeSearch",
		Episodes: mediaItemIds,
	}

	// send request
	resp, err := web.GetResponse(web.POST, web.JoinURL(p.apiUrl, "/command"), p.timeout, p.reqHeaders,
		&pvrDefaultRetry, req.BodyJSON(&payload))
	if err != nil {
		return false, errors.WithMessage(err, "failed retrieving command api response from whisparr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 201 {
		return false, fmt.Errorf("failed retrieving valid command api response from whisparr: %s",
			resp.Response().Status)
	}

	// decode response
	var q WhisparrV2CommandResponse
	if err := resp.ToJSON(&q); err != nil {
		return false, errors.WithMessage(err, "failed decoding command api response from whisparr")
	}

	// monitor search status
	p.log.WithField("command_id", q.Id).Debug("Monitoring search status")

	for {
		// retrieve command status
		searchStatus, err := p.getCommandStatus(q.Id)
		if err != nil {
			return false, errors.Wrapf(err, "failed retrieving command status from whisparr for: %d", q.Id)
		}

		p.log.WithFields(logrus.Fields{
			"command_id": q.Id,
			"status":     searchStatus.Status,
		}).Debug("Status retrieved")

		// is status complete?
		if searchStatus.Status == "completed" {
			break
		} else if searchStatus.Status == "failed" {
			return false, fmt.Errorf("search failed with message: %q", searchStatus.Message)
		} else if searchStatus.Status != "started" && searchStatus.Status != "queued" {
			return false, fmt.Errorf("search failed with unexpected status %q, message: %q", searchStatus.Status, searchStatus.Message)
		}

		time.Sleep(10 * time.Second)
	}

	return true, nil
}
