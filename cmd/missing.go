package cmd

import (
	"fmt"
	"github.com/l3uddz/wantarr/config"
	"github.com/l3uddz/wantarr/database"
	pvrObj "github.com/l3uddz/wantarr/pvr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tommysolsen/capitalise"
	"strings"
	"time"
)

var (
	maxQueueSize int
)

var missingCmd = &cobra.Command{
	Use:   "missing [PVR]",
	Short: "Search for missing media files",
	Long:  `This command can be used to search for missing media files from the respective arr wanted list.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate inputs
		if err := parseValidateInputs(args); err != nil {
			log.WithError(err).Fatal("Failed validating inputs")
		}

		// init pvr object
		if err := pvr.Init(); err != nil {
			log.WithError(err).Fatalf("Failed initializing pvr object for: %s", pvrName)
		}

		// load database
		if err := database.Init(flagDatabaseFile); err != nil {
			log.WithError(err).Fatal("Failed opening database file")
		}
		defer database.Close()

		// retrieve missing records from pvr and stash in database
		existingItemsCount := database.GetItemsCount(lowerPvrName, "missing")
		if flagRefreshCache || existingItemsCount < 1 {
			log.Infof("Retrieving missing media from %s: %q", capitalise.First(pvrConfig.Type), pvrName)

			missingRecords, err := pvr.GetWantedMissing()
			if err != nil {
				log.WithError(err).Fatal("Failed retrieving wanted missing pvr items...")
			}

			// stash missing media in database
			log.Debug("Stashing media items in database...")

			if err := database.SetMediaItems(lowerPvrName, "missing", missingRecords); err != nil {
				log.WithError(err).Fatal("Failed stashing media items in database")
			}

			log.Info("Stashed media items")

			// remove media no longer missing
			if existingItemsCount >= 1 {
				log.Debug("Removing media items from database that are no longer missing...")

				removedItems, err := database.DeleteMissingItems(lowerPvrName, "missing", missingRecords)
				if err != nil {
					log.WithError(err).Fatal("Failed removing media items from database that are no longer missing...")
				}

				log.WithField("removed_items", removedItems).
					Info("Removed media items from database that are no longer missing")
			}
		}

		// start queue monitor

		// get media items from database
		mediaItems, err := database.GetMediaItems(lowerPvrName, "missing")
		if err != nil {
			log.WithError(err).Fatal("Failed retrieving media items from database...")
		}
		log.WithField("media_items", len(mediaItems)).Debug("Retrieved media items from database")

		// start searching
		var searchItems []pvrObj.MediaItem
		batchSize := 10

		for _, item := range mediaItems {
			// add item to batch
			searchItems = append(searchItems, pvrObj.MediaItem{
				ItemId:     item.Id,
				AirDateUtc: item.AirDateUtc,
			})

			// not enough items batched yet
			if len(searchItems) < batchSize {
				continue
			}

			// generate slice of search item ids
			var searchItemIds []int
			for _, searchItem := range searchItems {
				searchItemIds = append(searchItemIds, searchItem.ItemId)
			}

			// do search
			searchTime := time.Now().UTC()
			ok, err := pvr.SearchMediaItems(searchItemIds)
			if err != nil {
				log.WithError(err).Fatal("Failed searching for items")
			} else if !ok {
				log.Fatal("Failed searching for items!")
			} else {
				log.Info("Searched for items!")

				// update search items with lastsearch time
				for pos, _ := range searchItems {
					(&searchItems[pos]).LastSearch = searchTime
				}

				if err := database.SetMediaItems(lowerPvrName, "missing", searchItems); err != nil {
					log.WithError(err).Fatal("Failed updating search items in database")
				}
			}

			// reset batch
			searchItems = []pvrObj.MediaItem{}
		}

	},
}

func init() {
	rootCmd.AddCommand(missingCmd)

	missingCmd.Flags().IntVarP(&maxQueueSize, "queue-size", "q", 5, "Exit when queue size reached.")
	missingCmd.Flags().BoolVarP(&flagRefreshCache, "refresh-cache", "r", false, "Refresh the locally stored cache.")
}

func parseValidateInputs(args []string) error {
	var ok bool = false
	var err error = nil

	// validate pvr exists in config
	pvrName = args[0]
	lowerPvrName = strings.ToLower(pvrName)
	pvrConfig, ok = config.Config.Pvr[pvrName]
	if !ok {
		return fmt.Errorf("no pvr configuration found for: %q", pvrName)
	}

	// init pvrObj
	pvr, err = pvrObj.Get(pvrName, pvrConfig.Type, pvrConfig)
	if err != nil {
		return errors.WithMessage(err, "failed loading pvr object")
	}

	return nil
}
