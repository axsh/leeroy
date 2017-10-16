package dropbox

import (
	"os"
	"testing"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
)

func TestFolderPoll(t *testing.T) {
	token, exists := os.LookupEnv("DROPBOX_TOKEN")
	if !exists {
		t.Skip("Unable to find DROPBOX_TOKEN env variable")
	}
	config := dropbox.Config{
		Token: token,
	}
	w := New("/polltest", config)
	if err := w.PollFolder(); err != nil {
		t.Error(err)
	}
}
