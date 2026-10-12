// Package include adds, updates and removes the marked block that points the user's ~/.ssh/config and
// ~/.gitconfig at the managed files. Everything outside the block is left byte for byte.
package include

import (
	"errors"
	"strings"
)

const (
	Begin = "# --- idset managed ---"
	End   = "# --- idset managed end ---"
)

// ErrUnterminated is returned when a file has the begin marker but no end marker after it; it is not touched.
var ErrUnterminated = errors.New("the idset managed block has no end marker")

// SSHBlock is the block for ~/.ssh/config.
func SSHBlock(managed string) string {
	return Begin + "\nInclude \"" + managed + "\"\n" + End + "\n"
}

// GitBlock is the block for ~/.gitconfig.
func GitBlock(managed string) string {
	return Begin + "\n[include]\n\tpath = " + managed + "\n" + End + "\n"
}

// find returns the start and end offsets of the first block, the end after its newline.
func find(content string) (start, end int, found bool, err error) {
	start = strings.Index(content, Begin)
	if start < 0 {
		return 0, 0, false, nil
	}
	rel := strings.Index(content[start:], End)
	if rel < 0 {
		return 0, 0, false, ErrUnterminated
	}
	end = start + rel + len(End)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return start, end, true, nil
}

// EnsureTop puts block at the start of content, where ssh reads the first matching Host, followed by a blank
// line. An existing block is replaced in place when it differs.
func EnsureTop(content, block string) (string, bool, error) {
	if out, done, changed, err := replace(content, block); done || err != nil {
		return out, changed, err
	}
	switch {
	case content == "":
		return block, true, nil
	case strings.HasPrefix(content, "\n"):
		return block + content, true, nil
	}
	return block + "\n" + content, true, nil
}

// EnsureBottom puts block at the end of content, where git lets it override earlier values, after a blank line.
// An existing block is replaced in place when it differs.
func EnsureBottom(content, block string) (string, bool, error) {
	if out, done, changed, err := replace(content, block); done || err != nil {
		return out, changed, err
	}
	switch {
	case content == "":
		return block, true, nil
	case strings.HasSuffix(content, "\n"):
		return content + "\n" + block, true, nil
	}
	return content + "\n\n" + block, true, nil
}

func replace(content, block string) (out string, done, changed bool, err error) {
	start, end, found, err := find(content)
	if err != nil {
		return content, true, false, err
	}
	if !found {
		return content, false, false, nil
	}
	if content[start:end] == block {
		return content, true, false, nil
	}
	return content[:start] + block + content[end:], true, true, nil
}

// Strip removes the first block and the blank lines right after it. When the block ended the file, the blank
// line EnsureBottom put before it goes too.
func Strip(content string) (string, bool, error) {
	start, end, found, err := find(content)
	if err != nil || !found {
		return content, false, err
	}
	for end < len(content) && content[end] == '\n' {
		end++
	}
	before, after := content[:start], content[end:]
	if after == "" && strings.HasSuffix(before, "\n\n") {
		before = before[:len(before)-1]
	}
	return before + after, true, nil
}
