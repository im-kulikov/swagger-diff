package jobs

import (
	"runtime"
)

// Parents worker
func (w *Worker) Parents() error {
	defer w.Info("done parents")
	defer close(w.changes)

	// iParents := 0

	for job := range w.parents {

		for _, fp := range job.Patches {
			from, to := fp.Files()

			if !checkFile("swagger.yaml", from, to) {
				continue
			}

			diffContent := formatDiff(fp.Chunks())
			fileBefore := contents(job.Commit, from, to)
			fileAfter := contents(job.Parent, from, to)

			// w.Infof("send changes: %s/%s", hash, job.Commit.Hash)
			w.changes <- &Commit{
				Author: job.Commit.Author.Name,
				Diff:   "```diff\n" + diffContent + "\n```",
				Time:   job.Commit.Author.When,
				URL:    formatDiffURL(w.repo, job.Parent.Hash, job.Commit.Hash, from, to),
				Hash:   job.Commit.Hash,
				Before: fileBefore,
				After:  fileAfter,
			}

			runtime.Gosched()
		}
	}

	return nil
}
