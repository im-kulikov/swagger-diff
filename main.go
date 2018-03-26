package main

import (
	"context"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/cryptopay-dev/yaga/logger/zap"
	"github.com/im-kulikov/swagger-diff/internal/jobs"
	"github.com/im-kulikov/swagger-diff/internal/swagger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

var log = zap.New(zap.Development)

func checkError(err error, message string, v ...interface{}) {
	if err != nil {
		log.Fatal(errors.Wrapf(err, message, v...))
	}
}
func findRepo() string {
	p := os.Getenv("GOPATH")
	items := strings.Split(p, ":")
	for _, item := range items {
		item = path.Join(item, "src", repoName)
		if file, err := os.Open(item); err == nil {
			return file.Name()
		}
	}

	log.Panic("can't find repo")
	return ""
}

func recoverPanic() {
	if err := recover(); err != nil {
		log.Fatal(err)
	}
}

var repoName = "github.com/<user>/<repo>"

func main() {
	repoPath := findRepo()
	r, err := git.PlainOpen(repoPath)
	checkError(err, "can't open")

	defer func(dt time.Time) {
		log.Print(time.Since(dt))
	}(time.Now())

	opts := jobs.Options{
		Logger:  log,
		Commits: make(chan *object.Commit, 100),
		Parents: make(chan *jobs.Job, 100),
		Changes: make(chan *jobs.Commit, 100),
		Result:  make([]*jobs.Commit, 0, 100),
		Repo:    repoName,
		Visited: jobs.CheckVisited(),
	}

	worker := jobs.New(opts)

	ref, err := r.Head()
	checkError(err, "can't get head")

	group, _ := errgroup.WithContext(context.Background())

	var head *object.Commit

	head, err = r.CommitObject(ref.Hash())
	checkError(err, "can't get commit objects")

	logs, err := r.Log(&git.LogOptions{From: head.Hash})
	checkError(err, "can't get logs")

	// Walk around changes:
	group.Go(func() error {
		defer log.Info("done changes")

		for commit := range opts.Changes {
			opts.Result = append(opts.Result, commit)
		}

		return nil
	})

	// Walk around parents:
	group.Go(worker.Parents)

	// Walk around commits:
	// group.Go(worker.Commits)

	// Receive commits:
	group.Go(func() error {
		defer close(opts.Parents)
		defer recoverPanic()

		errLog := logs.ForEach(func(commit *object.Commit) error {
			defer recoverPanic()

			if opts.Visited(commit.Hash.String()) {
				return nil
			}
			// log.Infof("VISIT: %s -> %d", commit.Hash, commit.NumParents())

			errParents := commit.Parents().ForEach(func(parent *object.Commit) error {
				defer recoverPanic()

				if opts.Visited(parent.Hash.String()) {
					return nil
				}

				// log.Infof("VISIT: %s -> %s", commit.Hash, parent.Hash)

				patch, errPatch := commit.Patch(parent)
				if errPatch != nil {
					log.Infof("can't get patch: %v", errPatch)
					return nil
				}

				opts.Parents <- &jobs.Job{
					Commit:  commit,
					Parent:  parent,
					Patches: patch.FilePatches(),
				}

				return nil
			})

			if errParents != nil {
				log.Infof("fetch parents: %v", errParents)
			}

			return nil
		})

		if errLog != nil {
			log.Infof("fetch logs: %v", errLog)
		}

		return nil
	})

	err = group.Wait()
	checkError(err, "after wait")

	sort.Slice(opts.Result, func(i, j int) bool {
		return opts.Result[i].Time.Before(opts.Result[j].Time)
	})

	tpl, err := template.New("main").Funcs(map[string]interface{}{
		"last": func(x int) bool {
			return len(opts.Result) == (x - 1)
		},
		"format": func(x time.Time) string {
			return x.Format(time.RFC3339)
		},
	}).Parse(tplContent)
	checkError(err, "template")

	fileOut := "/tmp/CHANGELOG.md"
	out, err := os.Create(fileOut)
	checkError(err, "can't create(%s)", fileOut)

	size := len(opts.Result)
	for i, commit := range opts.Result {
		if _, err = swagger.LoadContent(commit.Before); err != nil {
			log.Infof("[%02d / %02d] can't load before-content(%s):\n%v", i+1, size, commit.Hash.String(), err)
		}

		if _, err = swagger.LoadContent(commit.After); err != nil {
			log.Infof("[%02d / %02d] can't load after-content(%s):\n%v", i+1, size, commit.Hash.String(), err)
		}

		commit.After = "```yaml\n" + commit.After + "\n```"
		commit.Before = "```yaml\n" + commit.Before + "\n```"
	}

	defer log.Info(len(opts.Result))

	err = tpl.Execute(out, opts.Result)
	checkError(err, "can't execute template")
}

const tplContent = `
# Changelog Swagger:
{{$n := len .}}{{range $i, $Commit := .}}
## [{{ format $Commit.Time }}]({{ $Commit.URL }})

<details>
<summary>Show diff</summary>

{{ $Commit.Diff }}
</details>

{{ if (last $i) }} --- {{end}}

{{end}}
`

// <details>
// <summary>Show before</summary>
//
// {{ $Commit.Before }}
// </details>
//
// <details>
// <summary>Show after</summary>
//
// {{ $Commit.After }}
// </details>
