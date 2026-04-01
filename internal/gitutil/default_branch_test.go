package gitutil

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	outputs map[string][]byte
	runErrs map[string]error
}

var errMissingCommand = errors.New("missing command")

func (f fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	if out, ok := f.outputs[commandKey(name, args...)]; ok {
		return out, nil
	}
	return nil, errMissingCommand
}

func (f fakeRunner) CombinedOutput(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	return f.Output(context.Background(), "", name, args...)
}

func (f fakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	if err, ok := f.runErrs[commandKey(name, args...)]; ok {
		return err
	}
	return errMissingCommand
}

func TestDefaultBranchWithRunner_UsesOriginHead(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string][]byte{
			commandKey("git", "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"): []byte("refs/remotes/origin/main\n"),
		},
	}

	got := defaultBranchWithRunner(context.Background(), "/repo", runner)
	if got != "main" {
		t.Fatalf("defaultBranchWithRunner() = %q, want main", got)
	}
}

func TestDefaultBranchWithRunner_FallsBackToRemoteMain(t *testing.T) {
	runner := fakeRunner{
		runErrs: map[string]error{
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"): nil,
		},
	}

	got := defaultBranchWithRunner(context.Background(), "/repo", runner)
	if got != "main" {
		t.Fatalf("defaultBranchWithRunner() = %q, want main", got)
	}
}

func TestDefaultBranchWithRunner_FallsBackToLocalMaster(t *testing.T) {
	missing := errors.New("missing")
	runner := fakeRunner{
		runErrs: map[string]error{
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"):   missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/heads/main"):            missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/master"): missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/heads/master"):          nil,
		},
	}

	got := defaultBranchWithRunner(context.Background(), "/repo", runner)
	if got != "master" {
		t.Fatalf("defaultBranchWithRunner() = %q, want master", got)
	}
}

func TestDefaultBranchWithRunner_DefaultsToMain(t *testing.T) {
	missing := errors.New("missing")
	runner := fakeRunner{
		runErrs: map[string]error{
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"):   missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/heads/main"):            missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/master"): missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/heads/master"):          missing,
		},
	}

	got := defaultBranchWithRunner(context.Background(), "/repo", runner)
	if got != "main" {
		t.Fatalf("defaultBranchWithRunner() = %q, want main", got)
	}
}

func TestComparisonBaseWithRunner_PrefersRemoteTrackingRef(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string][]byte{
			commandKey("git", "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"): []byte("refs/remotes/origin/main\n"),
		},
		runErrs: map[string]error{
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"): nil,
		},
	}

	got := comparisonBaseWithRunner(context.Background(), "/repo", "", runner)
	if got != "origin/main" {
		t.Fatalf("comparisonBaseWithRunner() = %q, want origin/main", got)
	}
}

func TestComparisonBaseWithRunner_FallsBackToLocalBranch(t *testing.T) {
	missing := errors.New("missing")
	runner := fakeRunner{
		runErrs: map[string]error{
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"): missing,
			commandKey("git", "show-ref", "--verify", "--quiet", "refs/heads/main"):          nil,
		},
	}

	got := comparisonBaseWithRunner(context.Background(), "/repo", "origin/main", runner)
	if got != "main" {
		t.Fatalf("comparisonBaseWithRunner() = %q, want main", got)
	}
}

func commandKey(name string, args ...string) string {
	return strings.TrimSpace(name + " " + strings.Join(args, " "))
}
