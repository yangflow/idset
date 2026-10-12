// Package sysexec runs the system's git and ssh commands: with a timeout, no standard input, arguments as a list
// and never through a shell. Extra environment variables are added to the program's own.
package sysexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ErrTimeout is returned when a command runs past its timeout and is killed.
var ErrTimeout = errors.New("command timed out")

// Result is what a command printed and how it exited.
type Result struct {
	Stdout, Stderr string
	ExitCode       int
}

// Run runs name with args. A non-zero exit is not an error: it is in Result.ExitCode. The error is for a command
// that could not start or did not finish in time.
func Run(ctx context.Context, timeout time.Duration, env []string, name string, args ...string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("%s: %w", name, ErrTimeout)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		res.ExitCode = exit.ExitCode()
		return res, nil
	}
	return res, err
}
