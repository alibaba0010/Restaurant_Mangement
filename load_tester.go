package main

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	tests := []struct {
		name        string
		url         string
		concurrency int
		duration    time.Duration
	}{
		{
			name:        "Health Check (Public)",
			url:         "http://localhost:8001/api/v1/healthcheck",
			concurrency: 50,
			duration:    10 * time.Second,
		},
		{
			name:        "List Menus (Public)",
			url:         "http://localhost:8001/api/v1/menus",
			concurrency: 30,
			duration:    10 * time.Second,
		},
		{
			name:        "Swagger Docs (Public)",
			url:         "http://localhost:8001/swagger/index.html",
			concurrency: 10,
			duration:    5 * time.Second,
		},
	}

	fmt.Println("# API Load Test Results")
	fmt.Printf("Generated on: %s\n\n", time.Now().Format(time.RFC1123))

	for _, tt := range tests {
		runTest(tt.name, tt.url, tt.concurrency, tt.duration)
	}
}

func runTest(name, url string, concurrency int, duration time.Duration) {
	fmt.Printf("## %s\n", name)
	fmt.Printf("- URL: %s\n", url)
	fmt.Printf("- Concurrency: %d\n", concurrency)
	fmt.Printf("- Duration: %s\n\n", duration)

	results := make([]time.Duration, 0, 10000)
	var errorsCount uint64
	var requestsCount uint64
	var totalDuration int64 // in nanoseconds
	var mu sync.Mutex

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Timeout: 5 * time.Second,
			}
			for time.Since(start) < duration {
				atomic.AddUint64(&requestsCount, 1)
				reqStart := time.Now()
				resp, err := client.Get(url)
				reqDuration := time.Since(reqStart)

				if err != nil {
					atomic.AddUint64(&errorsCount, 1)
					continue
				}
				
				if resp.StatusCode >= 400 {
					atomic.AddUint64(&errorsCount, 1)
				}
				resp.Body.Close()

				mu.Lock()
				results = append(results, reqDuration)
				totalDuration += reqDuration.Nanoseconds()
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(start)

	mu.Lock()
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})
	numResults := len(results)
	mu.Unlock()

	happyResults := int(requestsCount) - int(errorsCount)
	successRate := float64(happyResults) / float64(requestsCount) * 100

	if numResults == 0 {
		fmt.Printf("No results collected (Requests: %d, Errors: %d)\n\n", requestsCount, errorsCount)
		return
	}

	p50 := results[numResults*50/100]
	p90 := results[numResults*90/100]
	p99 := results[int(float64(numResults)*0.99)]
	avg := time.Duration(totalDuration / int64(numResults))
	rps := float64(requestsCount) / actualDuration.Seconds()

	fmt.Println("| Metric | Value |")
	fmt.Println("| :--- | :--- |")
	fmt.Printf("| **Total Requests** | %d |\n", requestsCount)
	fmt.Printf("| **Success (2xx)** | %d |\n", happyResults)
	fmt.Printf("| **Errors (4xx/5xx/Net)** | %d |\n", errorsCount)
	fmt.Printf("| **Success Rate** | %.2f%% |\n", successRate)
	fmt.Printf("| **Average Latency** | %v |\n", avg)
	fmt.Printf("| **P50 (Median)** | %v |\n", p50)
	fmt.Printf("| **P90** | %v |\n", p90)
	fmt.Printf("| **P99** | %v |\n", p99)
	fmt.Printf("| **Throughput** | %.2f req/sec |\n", rps)
	fmt.Println("\n---")
}
