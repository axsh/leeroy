package github

import (
	"context"
	"fmt"

	"github.com/dropbox/godropbox/container/set"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type RepoChange struct {
	NewRefs     []*github.Reference
	RemovedRefs []*github.Reference
	UpdatedRefs []*github.Reference
}

type GithubWatcher struct {
	rootCtx  context.Context
	client   *github.Client
	lastRefs map[string]*github.Reference
}

func NewGithubWatcher(ghToken string) *GithubWatcher {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GithubWatcher{
		rootCtx:  ctx,
		client:   client,
		lastRefs: make(map[string]*github.Reference),
	}
}

func (w *GithubWatcher) PollRepository(owner, repo string, stopCh <-chan interface{}) (<-chan *RepoChange, <-chan error) {
	ch := make(chan *RepoChange, 1)
	errCh := make(chan error, 1)
	go func() {
		for {
			select {
			case <-stopCh:
				close(ch)
				return
			default:
				err := w.pollOneshot(owner, repo, func(n, r, u []*github.Reference) {
					ch <- &RepoChange{
						NewRefs:     n,
						RemovedRefs: r,
						UpdatedRefs: u,
					}
				})
				if err != nil {
					errCh <- err
					return
				}
			}
		}
	}()
	return ch, errCh
}

func (w *GithubWatcher) pollOneshot(owner, repo string, cb func(newRefs, removedRefs, updatedRefs []*github.Reference)) error {
	refs, res, err := w.client.Git.ListRefs(w.rootCtx, owner, repo, nil)
	if err != nil {
		return err
	}
	if !(200 >= res.StatusCode && res.StatusCode <= 299) {
		return fmt.Errorf("Invalid HTTP response: %d %s", res.StatusCode, res.Status)
	}
	remoteRefs := make(map[string]*github.Reference)
	remoteSet := set.NewSet()
	for _, r := range refs {
		remoteSet.Add(r.GetRef())
		remoteRefs[r.GetRef()] = r
	}
	lastSet := set.NewSet()
	for k, _ := range w.lastRefs {
		lastSet.Add(k)
	}

	updatedRefs := []*github.Reference{}
	set.Intersect(lastSet, remoteSet).Do(func(v interface{}) {
		rRef := remoteRefs[v.(string)]
		lRef := w.lastRefs[v.(string)]
		if rRef.Object.GetSHA() != lRef.Object.GetSHA() {
			updatedRefs = append(updatedRefs, rRef)
		}
	})
	newRefs := []*github.Reference{}
	set.Subtract(remoteSet, lastSet).Do(func(v interface{}) {
		ref := v.(string)
		newRefs = append(newRefs, remoteRefs[ref])
	})
	removedRefs := []*github.Reference{}
	set.Subtract(lastSet, remoteSet).Do(func(v interface{}) {
		ref := v.(string)
		removedRefs = append(removedRefs, w.lastRefs[ref])
	})
	if cb != nil {
		cb(newRefs, removedRefs, updatedRefs)
	}
	w.lastRefs = remoteRefs
	return nil
}
