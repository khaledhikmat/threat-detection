package main

import (
	"context"
	"fmt"

	"github.com/khaledhikmat/threat-detection-shared/models"
)

func elastic(_ context.Context, clip models.RecordingClip) error {

	fmt.Printf("elastic media indexer received a recording clip - TYPE %s - CLOUD REF %s - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedMediaIndexType(), clip.CloudReference, clip.StorageProvider, clip.Capturer, clip.Camera)

	// Store clip in database
	err := persistenceSvc.NewClip(clip)
	if err != nil && err.Error() != "IGNORE error" {
		fmt.Printf("elastic media indexer failed to store clip in elastic - %s\n", err.Error())
		return err
	}

	return nil
}
