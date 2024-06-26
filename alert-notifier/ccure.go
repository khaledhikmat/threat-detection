package main

import (
	"context"
	"fmt"
	"time"

	"github.com/khaledhikmat/threat-detection-shared/models"
)

func ccure(ctx context.Context, clip models.RecordingClip) error {

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

	// Indicate the alert invocation has ended
	clip.AlertInvocationBeginTime = time.Now()

	return nil
}
