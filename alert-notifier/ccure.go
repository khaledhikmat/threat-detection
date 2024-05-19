package main

import (
	"context"
	"fmt"

	"github.com/khaledhikmat/threat-detection-shared/equates"
)

func ccure(ctx context.Context, clip equates.RecordingClip) error {

	// Retrieve the recording clip from storage
	b, err := storageSvc.RetrieveRecordingClip(ctx, clip)
	if err != nil {
		fmt.Println("Failed to retrieve event's clip", err)
		return err
	}

	fmt.Printf("ccure alert notifier received a recording clip - TYPE %s - CLOUD REF %s - BYTES %d - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedAlertType(), clip.CloudReference, len(b), clip.StorageProvider, clip.Capturer, clip.Camera)

	// TODO: Do invoke ccure and feed it a byte array
	// TODO: Also....alert to different locations based on time of day

	return nil
}
