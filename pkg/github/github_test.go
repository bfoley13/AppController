package github

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubService(t *testing.T) {
	t.Run("DownloadRepo", func(t *testing.T) {
		s := NewGitHubService("")
		fileBytes, err := s.DownloadRepo(context.Background(), "bfoley13", "go_echo", "main")
		assert.Nil(t, err)
		log.Println(fileBytes)
	})
}
