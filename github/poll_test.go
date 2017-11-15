package github

import (
	"os"
	"testing"
	"time"

	"github.com/google/go-github/github"
)

type testConfig struct {
	Token string
}

func configTestEnv(t *testing.T) *testConfig {
	token, exists := os.LookupEnv("GITHUB_TOKEN")
	if !exists {
		t.Skip("Unable to find GITHUB_TOKEN env variable")
	}
	return &testConfig{
		Token: token,
	}
}

func TestGithubWatcher_pollOneshot(t *testing.T) {
	cfg := configTestEnv(t)
	w := NewGithubWatcher(cfg.Token)
	passed := false
	err := w.pollOneshot("axsh", "leeroy", func(n, r, u []*github.Reference) {
		passed = true
	})
	if err != nil {
		t.Error("Failed poolOneshot: ", err)
	}
	if !passed {
		t.Error("Never passed the result callback.")
	}
}

func TestGithubWatcher_PollRepository(t *testing.T) {
	cfg := configTestEnv(t)
	w := NewGithubWatcher(cfg.Token)
	ch, errCh := w.PollRepository("axsh", "leeroy")
	time.Sleep(1 * time.Second)
	passed := false
	select {
	case <-ch:
		passed = true
	case err := <-errCh:
		t.Error("Received error: ", err)
	}
	if !passed {
		t.Error("Never received the response from Github")
	}
	w.Stop()

	select {
	case <-ch:
		// closed properly
	case <-time.After(5 * time.Second):
		t.Error("stopCh signal seemed to be ignored")
	}
}
