package service

import (
	"encoding/json"
	"log"

	"github.com/flamefks/scheduler-system/internal/fetcher/client"
	natsqueue "github.com/flamefks/scheduler-system/internal/queue/nats"
)

type FetchJob struct {
	JobID   string            `json:"job_id"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout"`
}

type FetcherService struct {
	httpClient *client.HTTPClient
	publisher  *natsqueue.Publisher
}

func NewFetcherService() *FetcherService {
	return &FetcherService{
		httpClient: client.NewHTTPClient(),
	}
}

func (f *FetcherService) Handle(data []byte) error {
	var job FetchJob

	if err := json.Unmarshal(data, &job); err != nil {
		return err
	}

	log.Println("Fetching:", job.URL)

	body, err := f.httpClient.Do(job.URL, job.Method, job.Timeout)
	if err != nil {
		return err
	}

	// TODO: schema validation

	// TODO: publish to delivery queue

	log.Println("Fetched data for job:", job.JobID)

	return nil
}
