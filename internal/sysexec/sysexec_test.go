package sysexec

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// The test binary doubles as the command under test, so the tests run the same on every platform.
func TestMain(m *testing.M) {
	switch os.Getenv("SYSEXEC_HELPER") {
	case "echo":
		os.Stdout.WriteString(os.Getenv("SYSEXEC_VALUE") + " " + strings.Join(os.Args[1:], ","))
		os.Stderr.WriteString("to stderr")
		os.Exit(3)
	case "sleep":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRunReportsOutputAndExitCode(t *testing.T) {
	res, err := Run(context.Background(), 10*time.Second,
		[]string{"SYSEXEC_HELPER=echo", "SYSEXEC_VALUE=hi"}, os.Args[0], "a b", "c")
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "hi a b,c" || res.Stderr != "to stderr" || res.ExitCode != 3 {
		t.Fatalf("%+v", res)
	}
}

func TestRunTimesOut(t *testing.T) {
	start := time.Now()
	_, err := Run(context.Background(), 200*time.Millisecond, []string{"SYSEXEC_HELPER=sleep"}, os.Args[0])
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err %v", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("the command was not killed")
	}
}

func TestRunMissingCommand(t *testing.T) {
	if _, err := Run(context.Background(), time.Second, nil, "idset-no-such-command"); err == nil || errors.Is(err, ErrTimeout) {
		t.Fatalf("err %v", err)
	}
}
