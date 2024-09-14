package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v42/github"
	retry "github.com/sethvargo/go-retry"
	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RepoTar struct {
	FileBytes []byte
}

func (r *RepoTar) Write(p []byte) (n int, err error) {
	if r.FileBytes == nil {
		r.FileBytes = []byte{}
	}

	r.FileBytes = append(r.FileBytes, p...)
	return len(p), nil
}

type GitHubService struct {
	client *github.Client
}

func NewGitHubService(token string) *GitHubService {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &GitHubService{
		client: github.NewClient(tc),
	}
}

func (g *GitHubService) DownloadRepo(ctx context.Context, owner, repo, branch string) ([]byte, error) {
	u := fmt.Sprintf("repos/%s/%s/tarball/%s", owner, repo, branch)
	req, err := g.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Accept", "application/vnd.github+json")

	repoTar := new(RepoTar)
	resp, err := g.client.Do(ctx, req, repoTar)
	if err != nil {
		return nil, err
	}

	if resp != nil && resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return repoTar.FileBytes, nil
}

func (g *GitHubService) CreateBranch(ctx context.Context, owner, repo, branch string) error {
	baseRef := &github.Reference{}
	err := retry.Do(ctx, retry.WithMaxRetries(3, retry.NewExponential(time.Millisecond*300)), func(ctx context.Context) error {
		var err error
		resp := &github.Response{}
		baseRef, resp, err = g.client.Git.GetRef(ctx, owner, repo, "refs/heads/main")
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return retry.RetryableError(fmt.Errorf("getting base branch ref: %w", err))
			}
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	newRef := &github.Reference{Ref: github.String("refs/heads/" + branch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	_, _, err = g.client.Git.CreateRef(ctx, owner, repo, newRef)
	if err != nil {
		return err
	}

	return nil
}

func (g *GitHubService) CreateFiles(ctx context.Context, owner, repo, branch, filePath string, content []byte) error {
	lgr := log.FromContext(ctx, "githubservice")
	opt := &github.RepositoryContentFileOptions{
		Branch:  github.String(branch),
		Message: github.String(fmt.Sprintf("creating file: %s", filePath)),
		Content: content,
	}
	err := retry.Do(ctx, retry.WithMaxRetries(3, retry.NewExponential(time.Millisecond*200)), func(ctx context.Context) error {
		_, resp, err := g.client.Repositories.CreateFile(ctx, owner, repo, filePath, opt)
		if err != nil {
			// the github client return an error object on 404
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return retry.RetryableError(err)
			}
			lgr.Info(fmt.Sprintf("unrecoverable github error: %s", err.Error()))
			return err
		}

		return nil
	})
	if err != nil {
		lgr.Info(fmt.Sprintf("failed to create new file: %s", err.Error()))
		return err
	}

	return nil
}
