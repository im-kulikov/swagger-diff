package hunks

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing/format/diff"
)

const (
	diffInit = "diff --git a/%s b/%s\n"

	chunkStart  = "@@ -"
	chunkMiddle = " +"
	chunkEnd    = " @@\n%s\n"
	chunkCount  = "%d,%d"

	noFilePath = "/dev/null"
	aDir       = "a/"
	bDir       = "b/"

	fPath  = "--- %s\n"
	tPath  = "+++ %s\n"
	binary = "Binary files %s and %s differ\n"

	addLine    = "+%s\n"
	deleteLine = "-%s\n"
	equalLine  = " %s\n"

	oldMode         = "old mode %o\n"
	newMode         = "new mode %o\n"
	deletedFileMode = "deleted file mode %o\n"
	newFileMode     = "new file mode %o\n"

	renameFrom     = "from"
	renameTo       = "to"
	renameFileMode = "rename %s %s\n"

	indexAndMode = "index %s..%s %o\n"
	indexNoMode  = "index %s..%s\n"

	DefaultContextLines = 3
)

type hunk struct {
	fromLine int
	toLine   int

	fromCount int
	toCount   int

	ctxPrefix string
	ops       []*op
}

func (c *hunk) WriteTo(buf *bytes.Buffer) {
	buf.WriteString(chunkStart)

	if c.fromCount == 1 {
		fmt.Fprintf(buf, "%d", c.fromLine)
	} else {
		fmt.Fprintf(buf, chunkCount, c.fromLine, c.fromCount)
	}

	buf.WriteString(chunkMiddle)

	if c.toCount == 1 {
		fmt.Fprintf(buf, "%d", c.toLine)
	} else {
		fmt.Fprintf(buf, chunkCount, c.toLine, c.toCount)
	}

	fmt.Fprintf(buf, chunkEnd, c.ctxPrefix)

	for _, d := range c.ops {
		buf.WriteString(d.String())
	}
}

func (c *hunk) AddOp(t diff.Operation, s ...string) {
	ls := len(s)
	switch t {
	case diff.Add:
		c.toCount += ls
	case diff.Delete:
		c.fromCount += ls
	case diff.Equal:
		c.toCount += ls
		c.fromCount += ls
	}

	for _, l := range s {
		c.ops = append(c.ops, &op{l, t})
	}
}

type op struct {
	text string
	t    diff.Operation
}

func (o *op) String() string {
	var prefix string
	switch o.t {
	case diff.Add:
		prefix = addLine
	case diff.Delete:
		prefix = deleteLine
	case diff.Equal:
		prefix = equalLine
	}

	return fmt.Sprintf(prefix, o.text)
}

type hunksGenerator struct {
	fromLine, toLine            int
	ctxLines                    int
	chunks                      []diff.Chunk
	current                     *hunk
	hunks                       []*hunk
	beforeContext, afterContext []string
}

func New(chunks []diff.Chunk) *hunksGenerator {
	return &hunksGenerator{
		chunks:   chunks,
		ctxLines: DefaultContextLines,
	}
}

func splitLines(s string) []string {
	out := strings.Split(s, "\n")
	if out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	return out
}

func (c *hunksGenerator) Generate() []*hunk {
	for i, chunk := range c.chunks {
		ls := splitLines(chunk.Content())
		lsLen := len(ls)

		switch chunk.Type() {
		case diff.Equal:
			c.fromLine += lsLen
			c.toLine += lsLen
			c.processEqualsLines(ls, i)
		case diff.Delete:
			if lsLen != 0 {
				c.fromLine++
			}

			c.processHunk(i, chunk.Type())
			c.fromLine += lsLen - 1
			c.current.AddOp(chunk.Type(), ls...)
		case diff.Add:
			if lsLen != 0 {
				c.toLine++
			}
			c.processHunk(i, chunk.Type())
			c.toLine += lsLen - 1
			c.current.AddOp(chunk.Type(), ls...)
		}

		if i == len(c.chunks)-1 && c.current != nil {
			c.hunks = append(c.hunks, c.current)
		}
	}

	return c.hunks
}

func (c *hunksGenerator) processHunk(i int, op diff.Operation) {
	if c.current != nil {
		return
	}

	var ctxPrefix string
	linesBefore := len(c.beforeContext)
	if linesBefore > c.ctxLines {
		ctxPrefix = " " + c.beforeContext[linesBefore-c.ctxLines-1]
		c.beforeContext = c.beforeContext[linesBefore-c.ctxLines:]
		linesBefore = c.ctxLines
	}

	c.current = &hunk{ctxPrefix: ctxPrefix}
	c.current.AddOp(diff.Equal, c.beforeContext...)

	switch op {
	case diff.Delete:
		c.current.fromLine, c.current.toLine =
			c.addLineNumbers(c.fromLine, c.toLine, linesBefore, i, diff.Add)
	case diff.Add:
		c.current.toLine, c.current.fromLine =
			c.addLineNumbers(c.toLine, c.fromLine, linesBefore, i, diff.Delete)
	}

	c.beforeContext = nil
}

// addLineNumbers obtains the line numbers in a new chunk
func (c *hunksGenerator) addLineNumbers(la, lb int, linesBefore int, i int, op diff.Operation) (cla, clb int) {
	cla = la - linesBefore
	// we need to search for a reference for the next diff
	switch {
	case linesBefore != 0 && c.ctxLines != 0:
		clb = lb - c.ctxLines + 1
	case c.ctxLines == 0:
		clb = lb - c.ctxLines
	case i != len(c.chunks)-1:
		next := c.chunks[i+1]
		if next.Type() == op || next.Type() == diff.Equal {
			// this diff will be into this chunk
			clb = lb + 1
		}
	}

	return
}

func (c *hunksGenerator) processEqualsLines(ls []string, i int) {
	if c.current == nil {
		c.beforeContext = append(c.beforeContext, ls...)
		return
	}

	c.afterContext = append(c.afterContext, ls...)
	if len(c.afterContext) <= c.ctxLines*2 && i != len(c.chunks)-1 {
		c.current.AddOp(diff.Equal, c.afterContext...)
		c.afterContext = nil
	} else {
		ctxLines := c.ctxLines
		if ctxLines > len(c.afterContext) {
			ctxLines = len(c.afterContext)
		}
		c.current.AddOp(diff.Equal, c.afterContext[:ctxLines]...)
		c.hunks = append(c.hunks, c.current)

		c.current = nil
		c.beforeContext = c.afterContext[ctxLines:]
		c.afterContext = nil
	}
}
