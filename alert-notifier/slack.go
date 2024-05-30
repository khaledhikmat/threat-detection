package main

import (
	"context"
	"fmt"
	"time"

	"github.com/khaledhikmat/threat-detection-shared/models"
)

func slack(ctx context.Context, clip models.RecordingClip) error {

	// Retrieve the recording clip from storage
	b, err := storageSvc.RetrieveRecordingClip(ctx, clip)
	if err != nil {
		fmt.Println("Failed to retrieve event's clip", err)
		return err
	}

	fmt.Printf("slack alert notifier received a recording clip - TYPE %s - CLOUD REF %s - BYTES %d - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedAlertType(), clip.CloudReference, len(b), clip.StorageProvider, clip.Capturer, clip.Camera)

	// TODO: Do invoke slack and feed it a byte array

	// Indicate the alert invocation has ended
	clip.AlertInvocationBeginTime = time.Now()

	return nil
}
