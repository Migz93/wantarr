package pvr

import (
	"fmt"
	"strings"
	"time"

	"github.com/jpillora/backoff"
	"github.com/migz93/wantarr/config"
	"github.com/migz93/wantarr/utils/web"
)

var (
	pvrDefaultPageSize = 1000
	pvrDefaultTimeout  = 120
	pvrDefaultRetry    = web.Retry{
		MaxAttempts: 6,
		RetryableStatusCodes: []int{
			504,
		},
		Backoff: backoff.Backoff{
			Jitter: true,
			Min:    500 * time.Millisecond,
			Max:    10 * time.Second,
		},
	}
)

type MediaItem struct {
	ItemId     int
	AirDateUtc time.Time
	LastSearch time.Time
}

type Interface interface {
	Init() error
	GetQueueSize() (int, error)
	GetWantedMissing() ([]MediaItem, error)
	GetWantedCutoff() ([]MediaItem, error)
	SearchMediaItems([]int) (bool, error)
}

/* Public */

func Get(pvrName string, pvrType string, pvrConfig *config.Pvr) (Interface, error) {
	switch strings.ToLower(pvrType) {
	case "sonarr_v3":
		return NewSonarrV3(pvrName, pvrConfig), nil
	case "sonarr_v4":
		return NewSonarrV4(pvrName, pvrConfig), nil
	case "radarr_v2":
		return NewRadarrV2(pvrName, pvrConfig), nil
	case "radarr_v3":
		return NewRadarrV3(pvrName, pvrConfig), nil
	case "radarr_v4":
		return NewRadarrV4(pvrName, pvrConfig), nil
	case "radarr_v5":
		return NewRadarrV5(pvrName, pvrConfig), nil
	default:
		break
	}

	return nil, fmt.Errorf("unsupported pvr type provided: %q", pvrType)
}
