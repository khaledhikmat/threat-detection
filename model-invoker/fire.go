package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/khaledhikmat/threat-detection-shared/equates"
	"github.com/khaledhikmat/threat-detection-shared/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func fire(ctx context.Context, clip equates.RecordingClip) error {

	// Retrieve the recording clip from storage
	b, err := storageSvc.RetrieveRecordingClip(ctx, clip)
	if err != nil {
		fmt.Println("Failed to retrieve event's clip", err)
		return err
	}

	fmt.Printf("fire model invoker received a recording clip - MODEL %s - CLOUD REF %s - BYTES %d - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedAIModel(), clip.CloudReference, len(b), clip.StorageProvider, clip.Capturer, clip.Camera)

	// TODO: Invoke the fire model and feed it a byte array

	// In the meantime....generate 0 ~ 20 random fire tags
	tags := utils.RandFireTags(rand.Intn(20))

	// Add the tags to the clip
	clip.Tags = tags
	clip.TagsCount = len(tags)

	// Check if the tags contain "fire" which means fire was detected
	if utils.Contains(tags, "fire") {
		clip.AlertsCount = 1
		// Publish to the alerts topic
		fmt.Printf("fire model invoker publishes alert: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
		err = publisherSvc.PublishRecordingClip(ctx, equates.ThreatDetectionPubSub, equates.AlertTopic, clip)
		if err != nil {
			fmt.Printf("fire model invoker is unable to publish event to the alert topic: %s %v\n", clip.LocalReference, err)
		}
	}

	// Always publish to the metadata topic
	fmt.Printf("fire model invoker publishes metadata: %s - rags: %d\n", clip.LocalReference, len(clip.Tags))
	err = publisherSvc.PublishRecordingClip(ctx, equates.ThreatDetectionPubSub, equates.MetadataTopic, clip)
	if err != nil {
		fmt.Printf("fire model invoker is unable to publish event to the metadata topic: %s %v\n", clip.LocalReference, err)
	}

	return nil
}
