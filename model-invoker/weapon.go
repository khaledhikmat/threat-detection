package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/khaledhikmat/threat-detection-shared/models"
	"github.com/khaledhikmat/threat-detection-shared/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func weapon(ctx context.Context, clip models.RecordingClip) error {
	fmt.Printf("weapon model invoker received a recording clip - MODEL %s - CLOUD REF %s - PROVIDER %s - CAPTURER %s - AGENT %s\n",
		configSvc.GetSupportedAIModel(), clip.CloudReference, clip.StorageProvider, clip.Capturer, clip.Camera)

	if os.Getenv("INVOKER_API") != "" {
		return invokeWeaponModelViaAPI(ctx, clip)
	}

	// Retrieve the recording clip from storage
	start := time.Now()
	_, err := storageSvc.RetrieveRecordingClip(ctx, clip)
	if err != nil {
		fmt.Println("Failed to retrieve event's clip", err)
		return err
	}
	fmt.Printf("Retrieved clip for weapon model invoker in %v\n", time.Since(start))

	// TODO: Invoke the weapon model and feed it a byte array

	// In the meantime....generate 0 ~ 20 random weapon tags
	tags := utils.RandWeaponTags(rand.Intn(20))

	// Add the tags to the clip
	clip.Tags = tags
	clip.TagsCount = len(tags)
	clip.ModelInvoker = "weapon"

	// Check if the tags contain "weapon" which means weapon was detected
	if utils.Contains(tags, "weapon") {
		clip.AlertsCount = 1
		clip.ClipType = 1 // Denote alert type
		// Publish to the alerts topic
		fmt.Printf("weapon model invoker publishes an alert: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
		// Indicate the model invocation has ended
		clip.ModelInvocationEndTime = time.Now()
		err = pubsubSvc.PublishRecordingClip(ctx, models.ThreatDetectionPubSub, alertsTopic, clip)
		if err != nil {
			fmt.Printf("weapon model invoker is unable to publish event to the alert topic: %s %v\n", clip.LocalReference, err)
		}
	}

	// Always publish to the metadata topic
	fmt.Printf("weapon model invoker publishes metadata: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
	clip.ClipType = 0 // Denote metadata type
	// Indicate the model invocation has ended
	clip.ModelInvocationEndTime = time.Now()
	err = pubsubSvc.PublishRecordingClip(ctx, models.ThreatDetectionPubSub, metadataTopic, clip)
	if err != nil {
		fmt.Printf("weapon model invoker is unable to publish event to the metadata topic: %s %v\n", clip.LocalReference, err)
	}

	return nil
}

// WARNING: Must match the Python API models
type weaponModelRequest struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type weaponModelResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func invokeWeaponModelViaAPI(ctx context.Context, clip models.RecordingClip) error {
	start := time.Now()

	apiClient := &http.Client{
		Transport: &headerRoundTripper{
			Next: &loggingRoundTripper{
				Next:   http.DefaultTransport,
				Logger: os.Stdout,
			},
		},
	}

	modelRequest := weaponModelRequest{
		ID:  clip.ID,
		URL: clip.CloudReference,
	}

	modelResponse := weaponModelResponse{}

	// TODO: Call the API
	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(&modelRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", os.Getenv("INVOKER_API"), payloadBuf)
	if err != nil {
		return err
	}

	res, err := apiClient.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return err
	}

	err = json.Unmarshal(body, &modelResponse)
	if err != nil {
		return err
	}

	fmt.Printf("Calling the fire model API took %v\n", time.Since(start))

	// In the meantime....generate 0 ~ 20 random weapon tags
	tags := utils.RandWeaponTags(rand.Intn(20))

	// Add the tags to the clip
	clip.Tags = tags
	clip.TagsCount = len(tags)
	clip.ModelInvoker = "weapon"

	// Check if the tags contain "weapon" which means weapon was detected
	if modelResponse.URL != "" {
		clip.AlertsCount = 1
		clip.ClipType = 1 // Denote alert type
		// Publish to the alerts topic
		fmt.Printf("weapon model invoker publishes an alert: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
		// Indicate the model invocation has ended
		clip.ModelInvocationEndTime = time.Now()
		err = pubsubSvc.PublishRecordingClip(ctx, models.ThreatDetectionPubSub, alertsTopic, clip)
		if err != nil {
			fmt.Printf("weapon model invoker is unable to publish event to the alert topic: %s %v\n", clip.LocalReference, err)
		}
	}

	// Always publish to the metadata topic
	fmt.Printf("weapon model invoker publishes metadata: %s - tags: %d\n", clip.LocalReference, len(clip.Tags))
	clip.ClipType = 0 // Denote metadata type
	// Indicate the model invocation has ended
	clip.ModelInvocationEndTime = time.Now()
	err = pubsubSvc.PublishRecordingClip(ctx, models.ThreatDetectionPubSub, metadataTopic, clip)
	if err != nil {
		fmt.Printf("weapon model invoker is unable to publish event to the metadata topic: %s %v\n", clip.LocalReference, err)
	}

	return nil
}
