package service

import (
	"context"
	"sync"

	"contrib.rocks/apps/api-go/core"
	"contrib.rocks/apps/api-go/infrastructure"
	"contrib.rocks/libs/goutils/model"
	"github.com/google/go-github/v45/github"
)

type GitHubService struct {
	c *infrastructure.GitHubClient
}

func NewGitHubService(i *core.Infrastructure) *GitHubService {
	return &GitHubService{i.GitHubClient}
}

func (s *GitHubService) GetContributors(ctx context.Context, r *model.Repository) (*model.RepositoryContributors, error) {
	type Result[T any] struct {
		Value T
		Error error
	}
	type RepositoryResult Result[*github.Repository]
	type ContributorsResult Result[[]*github.Contributor]

	wg := sync.WaitGroup{}
	wg.Add(2)
	repositoryChan := make(chan RepositoryResult, 1)
	contributorsChan := make(chan ContributorsResult, 1)
	go func(ch chan RepositoryResult) {
		defer wg.Done()
		data, err := s.fetchRepository(ctx, r)
		if err != nil {
			ch <- RepositoryResult{nil, err}
			return
		}
		ch <- RepositoryResult{data, nil}
		close(ch)
	}(repositoryChan)
	go func(ch chan ContributorsResult) {
		defer wg.Done()
		data, err := s.fetchContributors(ctx, r)
		if err != nil {
			ch <- ContributorsResult{nil, err}
			return
		}
		ch <- ContributorsResult{data, nil}
		close(ch)
	}(contributorsChan)
	wg.Wait()
	repositoryResult := <-repositoryChan
	contributorsResult := <-contributorsChan
	if repositoryResult.Error != nil {
		return nil, repositoryResult.Error
	}
	if contributorsResult.Error != nil {
		return nil, contributorsResult.Error
	}
	contributors := make([]*model.Contributor, 0, len(contributorsResult.Value))
	for _, e := range contributorsResult.Value {
		contributors = append(contributors, &model.Contributor{
			ID:            e.GetID(),
			Login:         e.GetLogin(),
			AvatarURL:     e.GetAvatarURL(),
			HTMLURL:       e.GetHTMLURL(),
			Contributions: e.GetContributions(),
		})
	}
	return &model.RepositoryContributors{
		Repository: &model.Repository{
			Owner:    repositoryResult.Value.Owner.GetLogin(),
			RepoName: repositoryResult.Value.GetName(),
		},
		StargazersCount: repositoryResult.Value.GetStargazersCount(),
		Contributors:    contributors,
	}, nil
}

func (s *GitHubService) fetchRepository(ctx context.Context, r *model.Repository) (*github.Repository, error) {
	ret, _, err := s.c.Repositories.Get(ctx, r.Owner, r.RepoName)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *GitHubService) fetchContributors(ctx context.Context, r *model.Repository) ([]*github.Contributor, error) {
	options := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var ret []*github.Contributor
	for {
		data, resp, err := s.c.Repositories.ListContributors(ctx, r.Owner, r.RepoName, options)
		if err != nil {
			return nil, err
		}
		ret = append(ret, data...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}
	return ret, nil
}