package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/khaledhikmat/threat-detection-shared/models"
	"github.com/khaledhikmat/threat-detection-shared/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func weapon(ctx context.Context, clip models.RecordingClip) error {

	// Retrieve the recording clip from storage
	b, err := storageSvc.RetrieveRecordingClip(ctx, clip)
	if err != nil {
		fmt.Println("Failed to retrieve event's clip", err)
		return err
	}

	fmt.Printf("weapon model invoker received a recording clip - MODEL %s - CLOUD REF %s - BYTES %d - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedAIModel(), clip.CloudReference, len(b), clip.StorageProvider, clip.Capturer, clip.Camera)

	// TODO: Invoke the fire model and feed it a byte array

	// In the meantime....generate 0 ~ 20 random weapon tags
	tags := utils.RandWeaponTags(rand.Intn(20))

	// Add the tags to the clip
	clip.Tags = tags
	clip.TagsCount = len(tags)

	// Check if the tags contain "fire" which means fire was detected
	if utils.Contains(tags, "weapon") {
		clip.AlertsCount = 1
		// Publish to the alerts topic
		fmt.Printf("weapon model invoker publishes an alert: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
		err = publisherSvc.PublishRecordingClip(ctx, models.ThreatDetectionPubSub, models.AlertTopic, clip)
		if err != nil {
			fmt.Printf("weapon model invoker is unable to publish event to the alert topic: %s %v\n", clip.LocalReference, err)
		}
	}

	// Always publish to the metadata topic
	fmt.Printf("weapon model invoker publishes metadata: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
	err = publisherSvc.PublishRecordingClip(ctx, models.ThreatDetectionPubSub, models.MetadataTopic, clip)
	if err != nil {
		fmt.Printf("weapon model invoker is unable to publish event to the metadata topic: %s %v\n", clip.LocalReference, err)
	}

	return nil
}
