package main

import (
	"context"
	"fmt"

	"github.com/khaledhikmat/threat-detection-shared/equates"
)

func database(_ context.Context, clip equates.RecordingClip) error {

	fmt.Printf("database media indexer received a recording clip - TYPE %s - CLOUD REF %s - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedMediaIndexType(), clip.CloudReference, clip.StorageProvider, clip.Capturer, clip.Camera)

	// Store clip in database
	err := persistenceSvc.NewClip(clip)
	if err != nil && err.Error() != "IGNORE error" {
		fmt.Printf("database media indexer failed to store clip in database - %s\n", err.Error())
		return err
	}

	return nil
}
