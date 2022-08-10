package usage

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/logging"
	"contrib.rocks/apps/api/internal/logger"
	"contrib.rocks/apps/api/internal/tracing"
	"contrib.rocks/libs/goutils/model"
)

type Service struct {
	usageLogger logger.Logger
}

func New(usageLogger logger.Logger) *Service {
	return &Service{usageLogger}
}

func (s *Service) CollectUsage(c context.Context, r *model.RepositoryContributors, via string) error {
	ctx, span := tracing.DefaultTracer.Start(c, "usage.Service.CollectUsage")
	defer span.End()

	s.usageLogger.Log(ctx, logging.Entry{
		Labels: map[string]string{
			"via": via,
		},
		Payload: struct {
			Repository   string `json:"repository"`
			Owner        string `json:"owner"`
			RepoName     string `json:"repo"`
			Stargazers   int    `json:"stargazers"`
			Contributors int    `json:"contributors"`
			Timestamp    string `json:"timestamp"`
		}{
			Repository:   r.Repository.String(), // TODO remove
			Owner:        r.Repository.Owner,
			RepoName:     r.Repository.RepoName,
			Stargazers:   r.StargazersCount,
			Contributors: len(r.Contributors),
			Timestamp:    fmt.Sprint(time.Now().UnixMilli()), // TODO remove
		},
	})

	return nil
}
