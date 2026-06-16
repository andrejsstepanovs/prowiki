package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestAppRunVersion(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := &App{
		Out: out,
		Err: errOut,
	}

	err := app.Run([]string{"--version"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := out.String()
	want := "prowiki version 0.1.0\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAppRunUnknownCommand(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := &App{
		Out: out,
		Err: errOut,
	}

	err := app.Run([]string{"foobar"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "unknown command: foobar") {
		t.Errorf("expected error about unknown command, got: %v", err)
	}
}

