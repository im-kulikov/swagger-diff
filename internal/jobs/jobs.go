package jobs

import (
	"time"

	"github.com/cryptopay-dev/yaga/logger"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/diff"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Worker struct
type Worker struct {
	logger.Logger
	visited visitedHandler
	commits chan *object.Commit
	parents chan *Job
	changes chan *Commit
	result  []*Commit
	repo    string
}

// Job struct
type Job struct {
	Commit  *object.Commit
	Parent  *object.Commit
	Patches []diff.FilePatch
}

// Commit struct
type Commit struct {
	Author  string
	Diff    string
	Time    time.Time
	URL     string
	Hash    plumbing.Hash
	After   string
	Before  string
	Content string
}

type visitedHandler func(string) bool

// Options of worker
type Options struct {
	Logger logger.Logger

	// Channels:
	Commits chan *object.Commit
	Parents chan *Job
	Changes chan *Commit

	// Helpers:
	Visited visitedHandler

	Repo string

	// Result:
	Result []*Commit
}

// New worker
func New(opts Options) *Worker {
	return &Worker{
		Logger:  opts.Logger,
		visited: opts.Visited,
		commits: opts.Commits,
		parents: opts.Parents,
		changes: opts.Changes,
		result:  opts.Result,
		repo:    opts.Repo,
	}
}
