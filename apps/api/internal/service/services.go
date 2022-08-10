package service

import (
	"io"

	"contrib.rocks/apps/api/internal/config"
	"contrib.rocks/apps/api/internal/logger"
	"contrib.rocks/apps/api/internal/service/internal/appcache"
	"contrib.rocks/apps/api/internal/service/internal/contributors"
	"contrib.rocks/apps/api/internal/service/internal/image"
	"contrib.rocks/apps/api/internal/service/internal/usage"
	"contrib.rocks/libs/goutils/apiclient"
)

type ServicePack struct {
	ContributorsService ContributorsService
	UsageService        UsageService
	ImageService        ImageService
	DefaultLogger       logger.Logger

	closables []io.Closer
}

func NewServicePack(cfg *config.Config) *ServicePack {
	sp := ServicePack{
		closables: []io.Closer{},
	}
	gh := apiclient.NewGitHubClient(cfg.GitHubAuthToken)

	var cache appcache.AppCache
	if cfg.ProjectID() != "" && cfg.CacheBucketName != "" {
		storageClient := apiclient.NewStorageClient()
		cache = appcache.NewGCSCache(storageClient, cfg.CacheBucketName)
	} else {
		cache = appcache.NewMemoryCache()
	}

	sp.DefaultLogger = logger.NewLogger("api/default")
	sp.ContributorsService = contributors.New(gh, cache, logger.NewLogger("contributors-json-cache-miss"))
	sp.ImageService = image.New(cache, logger.NewLogger("image-cache-miss"))
	sp.UsageService = usage.New(logger.NewLogger("repository-usage"))

	return &sp
}

func (sp *ServicePack) Close() {
	for _, fn := range sp.closables {
		fn.Close()
	}
}
