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

type ReadarrV0 struct {
	cfg        *config.Pvr
	log        *logrus.Entry
	apiUrl     string
	reqHeaders req.Header
	timeout    int
}

type ReadarrV0Queue struct {
	Size int `json:"totalRecords"`
}

type ReadarrV0Album struct {
	Id          int
	ReleaseDate time.Time
	Monitored   bool
}

type ReadarrV0Wanted struct {
	Page          int
	PageSize      int
	SortKey       string
	SortDirection string
	TotalRecords  int
	Records       []ReadarrV0Album
}

type ReadarrV0SystemStatus struct {
	Version string
}

type ReadarrV0CommandStatus struct {
	Name    string
	Message string
	Started time.Time
	Ended   time.Time
	Status  string
}

type ReadarrV0CommandResponse struct {
	Id int
}

type ReadarrV0BookSearch struct {
	Name  string `json:"name"`
	Books []int  `json:"BookIds"`
}

/* Initializer */

func NewReadarrV0(name string, c *config.Pvr) *ReadarrV0 {
	// set api url
	apiUrl := ""
	if strings.Contains(c.URL, "/api") {
		apiUrl = c.URL
	} else {
		apiUrl = web.JoinURL(c.URL, "/api/v1")
	}

	// set headers
	reqHeaders := req.Header{
		"X-Api-Key": c.ApiKey,
	}

	return &ReadarrV0{
		cfg:        c,
		log:        logger.GetLogger(name),
		apiUrl:     apiUrl,
		reqHeaders: reqHeaders,
		timeout:    pvrDefaultTimeout,
	}
}

/* Private */

func (p *ReadarrV0) getSystemStatus() (*ReadarrV0SystemStatus, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/system/status"), p.timeout, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving system status api response from readarr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid system status api response from readarr: %s",
			resp.Response().Status)
	}

	// decode response
	var s ReadarrV0SystemStatus
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding system status api response from readarr")
	}

	return &s, nil
}

func (p *ReadarrV0) getCommandStatus(id int) (*ReadarrV0CommandStatus, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, fmt.Sprintf("/command/%d", id)), p.timeout,
		p.reqHeaders, &pvrDefaultRetry)
	if err != nil {
		return nil, errors.New("failed retrieving command status api response from readarr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return nil, fmt.Errorf("failed retrieving valid command status api response from readarr: %s",
			resp.Response().Status)
	}

	// decode response
	var s ReadarrV0CommandStatus
	if err := resp.ToJSON(&s); err != nil {
		return nil, errors.WithMessage(err, "failed decoding command status api response from readarr")
	}

	return &s, nil
}

/* Interface Implements */

func (p *ReadarrV0) Init() error {
	// retrieve system status
	status, err := p.getSystemStatus()
	if err != nil {
		return errors.Wrap(err, "failed initializing readarr pvr")
	}

	// determine version
	switch status.Version[0:1] {
	case "0":
		break
	default:
		return fmt.Errorf("unsupported version of readarr pvr: %s", status.Version)
	}
	return nil
}

func (p *ReadarrV0) GetQueueSize() (int, error) {
	// send request
	resp, err := web.GetResponse(web.GET, web.JoinURL(p.apiUrl, "/queue"), p.timeout, p.reqHeaders,
		&pvrDefaultRetry)
	if err != nil {
		return 0, errors.WithMessage(err, "failed retrieving queue api response from readarr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 200 {
		return 0, fmt.Errorf("failed retrieving valid queue api response from readarr: %s",
			resp.Response().Status)
	}

	// decode response
	var q ReadarrV0Queue
	if err := resp.ToJSON(&q); err != nil {
		return 0, errors.WithMessage(err, "failed decoding queue api response from readarr")
	}

	p.log.WithField("queue_size", q.Size).Debug("Queue retrieved")
	return q.Size, nil
}

func (p *ReadarrV0) GetWantedMissing() ([]MediaItem, error) {
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
			return nil, errors.WithMessage(err, "failed retrieving wanted missing api response from readarr")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid wanted missing api response from readarr: %s",
				resp.Response().Status)
		}

		// decode response
		var m ReadarrV0Wanted
		if err := resp.ToJSON(&m); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding wanted missing api response from readarr")
		}

		// process response
		lastPageSize = len(m.Records)
		for _, episode := range m.Records {

			// store this episode
			airDate := episode.ReleaseDate
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

func (p *ReadarrV0) GetWantedCutoff() ([]MediaItem, error) {
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
			return nil, errors.WithMessage(err, "failed retrieving wanted cutotff unmet api response from readarr")
		}

		// validate response
		if resp.Response().StatusCode != 200 {
			_ = resp.Response().Body.Close()
			return nil, fmt.Errorf("failed retrieving valid wanted cutoff unmet api response from readarr: %s",
				resp.Response().Status)
		}

		// decode response
		var m ReadarrV0Wanted
		if err := resp.ToJSON(&m); err != nil {
			_ = resp.Response().Body.Close()
			return nil, errors.WithMessage(err, "failed decoding wanted cutoff unmet api response from readarr")
		}

		// process response
		lastPageSize = len(m.Records)
		for _, episode := range m.Records {
			// store this episode
			airDate := episode.ReleaseDate
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

func (p *ReadarrV0) SearchMediaItems(mediaItemIds []int) (bool, error) {
	// set request data
	payload := ReadarrV0BookSearch{
		Name:  "BookSearch",
		Books: mediaItemIds,
	}

	// send request
	resp, err := web.GetResponse(web.POST, web.JoinURL(p.apiUrl, "/command"), p.timeout, p.reqHeaders,
		&pvrDefaultRetry, req.BodyJSON(&payload))
	if err != nil {
		return false, errors.WithMessage(err, "failed retrieving command api response from readarr")
	}
	defer resp.Response().Body.Close()

	// validate response
	if resp.Response().StatusCode != 201 {
		return false, fmt.Errorf("failed retrieving valid command api response from readarr: %s",
			resp.Response().Status)
	}

	// decode response
	var q ReadarrV0CommandResponse
	if err := resp.ToJSON(&q); err != nil {
		return false, errors.WithMessage(err, "failed decoding command api response from readarr")
	}

	// monitor search status
	p.log.WithField("command_id", q.Id).Debug("Monitoring search status")

	for {
		// retrieve command status
		searchStatus, err := p.getCommandStatus(q.Id)
		if err != nil {
			return false, errors.Wrapf(err, "failed retrieving command status from readarr for: %d", q.Id)
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
