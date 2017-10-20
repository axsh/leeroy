package dropbox

import (
	"log"
	"net/http"
	"path"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
)

type noauthTransport struct {
	http.Transport
}

func (t *noauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Del("Authorization")
	return t.Transport.RoundTrip(req)
}

func newNoAuthClient() *http.Client {
	return &http.Client{
		Transport: &noauthTransport{},
	}
}

// Configuration parameters for leeroy.json
type Config struct {
	// Dropbox folder path to watch
	FolderPath string `json:folder_path`
	// Oauth2 access token
	Token string `json:"token"`
}

type DropboxWatcher struct {
	config   *Config
	filesApi files.Client
}

func NewWatcher(config *Config) (*DropboxWatcher, error) {
	c := dropbox.Config{
		Token: config.Token,
	}
	return &DropboxWatcher{
		config:   config,
		filesApi: files.New(c),
	}, nil
}

func (w *DropboxWatcher) iterateFolder() (*files.ListFolderResult, error) {
	req := files.NewListFolderArg(w.config.FolderPath)
	req.Recursive = true

	res, err := w.filesApi.ListFolder(req)
	if err != nil {
		return nil, err
	}
	for _, entry := range res.Entries {
		switch f := entry.(type) {
		case *files.FileMetadata:
			matched, err := path.Match(path.Join(w.config.FolderPath, "*", "rebuild.txt"), f.PathLower)
			if err != nil {
				return nil, err
			}
			if !matched {
				continue
			}
			branch := path.Base(path.Dir(f.PathLower))
			log.Print("Found rebuild.txt on branch:", branch)

			_, err = w.filesApi.DeleteV2(files.NewDeleteArg(f.PathLower))
			if err != nil {
				switch e := err.(type) {
				case files.DeleteAPIError:
					log.Print("Dropbox API Error: ", e)
					continue
				default:
					return nil, err
				}
			}
			log.Print("Removed rebuild.txt on branch: ", branch)
		}
	}
	return res, nil
}

func (w *DropboxWatcher) PollFolder() error {
	res, err := w.iterateFolder()
	if err != nil {
		return err
	}
	cursor := res.Cursor
	log.Printf("Start to poll '%s'", w.config.FolderPath)
	for {
		noauthdbx := files.New(dropbox.Config{Client: newNoAuthClient()})
		req := files.NewListFolderLongpollArg(cursor)
		res, err := noauthdbx.ListFolderLongpoll(req)
		if err != nil {
			return err
		}
		if !res.Changes {
			// re-use same cursor value
			continue
		}
		res2, err := w.iterateFolder()
		if err != nil {
			switch e := err.(type) {
			case files.ListFolderAPIError:
				log.Print("Dropbox API Error: ", e)
				continue
			default:
				return err
			}
		}
		cursor = res2.Cursor
	}
	return nil
}
