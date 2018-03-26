package jobs

import (
	"bytes"
	"crypto"
	"fmt"
	"strings"
	"sync"

	"github.com/im-kulikov/swagger-diff/internal/hunks"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/diff"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func md5hash(v string) []byte {
	hasher := crypto.MD5.New()
	_, err := hasher.Write([]byte(v))
	if err != nil {
		panic(err)
	}
	return hasher.Sum(nil)
}

// CheckVisited in map
func CheckVisited() func(string) bool {
	var visited = make(map[string]struct{})
	var locker sync.Mutex

	return func(hash string) bool {
		locker.Lock()
		defer locker.Unlock()

		if _, ok := visited[hash]; ok {
			return true
		}

		visited[hash] = struct{}{}
		return false
	}
}

func checkFile(name string, files ...diff.File) bool {
	for _, file := range files {
		if file == nil {
			continue
		}

		// log.Print(file.Path())
		if strings.Contains(file.Path(), name) {
			return true
		}
	}
	return false
}

func contents(commit *object.Commit, files ...diff.File) string {
	for _, file := range files {
		if file == nil {
			continue
		}

		f, err := commit.File(file.Path())
		if err != nil {
			continue
		}

		if data, err := f.Contents(); err == nil {
			return data
		}
	}

	return ""
}

func formatDiff(chunks []diff.Chunk) string {
	g := hunks.New(chunks)
	buf := new(bytes.Buffer)
	for _, c := range g.Generate() {
		c.WriteTo(buf)
	}
	return buf.String()
}

func formatDiffURL(repoName string, parent, current plumbing.Hash, from diff.File, to diff.File) string {
	var file string
	if from == nil && to == nil {
		return ""
	}

	if from != nil {
		file = from.Path()
	}

	if to != nil {
		file = to.Path()
	}

	return fmt.Sprintf(
		"https://%s/compare/%s...%s#diff-%x",
		repoName,
		parent,
		current,
		md5hash(file),
	)
}
