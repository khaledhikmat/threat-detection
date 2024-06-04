package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type headerRoundTripper struct {
	Next http.RoundTripper
}

func (b headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Content-Type", "application/json")
	return b.Next.RoundTrip(r)
}

type loggingRoundTripper struct {
	Next   http.RoundTripper
	Logger io.Writer
}

func (b loggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	//_, _ = b.Logger.Write([]byte("Request to " + r.URL.String() + "\n"))
	return b.Next.RoundTrip(r)
}

// WARNING: Must match the Python API models
type request struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type response struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func main() {
	iterations := 10
	url := "http://localhost:5003/detections"

	args := os.Args[1:]
	if len(args) > 0 {
		itr, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("arguments retrieval failed: %v\n", err)
			return
		}

		iterations = itr
	}
	if len(args) > 1 {
		url = args[1]
	}

	fmt.Printf("Model Exerciser against %s for %d iterations\n", url, iterations)

	start := time.Now()
	resultChan := make(chan response, iterations)
	wg := &sync.WaitGroup{}
	wg.Add(iterations)

	for i := 0; i < iterations; i++ {
		go caseRunner(wg, url, resultChan)
	}

	wg.Wait()
	close(resultChan)

	errors := 0
	predictions := 0
	results := []response{}
	for res := range resultChan {
		if res.ID != "" {
			fmt.Printf("Error => %s\n", res.ID)
			errors++
		}
		if res.URL != "" {
			predictions++
		}
		results = append(results, res)
	}

	fmt.Printf("Time taken %v - errors %d - matched predictions %d\n", time.Since(start), errors, predictions)
}

func caseRunner(wg *sync.WaitGroup, url string, result chan response) {
	defer wg.Done()

	apiClient := &http.Client{
		Transport: &headerRoundTripper{
			Next: &loggingRoundTripper{
				Next:   http.DefaultTransport,
				Logger: os.Stdout,
			},
		},
	}

	modelRequest := request{
		ID:  "",
		URL: "",
	}

	modelResponse := response{}

	// Call the model API
	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(&modelRequest)
	if err != nil {
		result <- response{
			ID:  err.Error(),
			URL: "",
		}
		return
	}

	req, err := http.NewRequest("POST", url, payloadBuf)
	if err != nil {
		result <- response{
			ID:  err.Error(),
			URL: "",
		}
		return
	}

	res, err := apiClient.Do(req)
	if err != nil {
		result <- response{
			ID:  err.Error(),
			URL: "",
		}
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		result <- response{
			ID:  err.Error(),
			URL: "",
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		result <- response{
			ID:  fmt.Sprintf("%d", res.StatusCode),
			URL: "",
		}
		return
	}

	err = json.Unmarshal(body, &modelResponse)
	if err != nil {
		result <- response{
			ID:  err.Error(),
			URL: "",
		}
		return
	}

	modelResponse.ID = ""
	result <- modelResponse
}
