package dropbox

import (
	"os"
	"testing"
)

func configTestEnv(t *testing.T) *Config {
	token, exists := os.LookupEnv("DROPBOX_TOKEN")
	if !exists {
		t.Skip("Unable to find DROPBOX_TOKEN env variable")
	}
	return &Config{
		FolderPath: "/polltest",
		Token:      token,
	}
}

func TestDropboxWatcher_NewWatcher(t *testing.T) {
	config := configTestEnv(t)
	_, err := NewWatcher(config)
	if err != nil {
		t.Error(err)
	}
}

func TestDropboxWatcher_FolderPoll(t *testing.T) {
	config := configTestEnv(t)
	w, err := NewWatcher(config)
	if err != nil {
		t.Error(err)
	}
	if err := w.PollFolder(); err != nil {
		t.Error(err)
	}
}
