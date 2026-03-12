package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIFlow(t *testing.T) {
	repo := initRepo(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	var out bytes.Buffer
	var errOut bytes.Buffer

	if err := run([]string{"init", "-p", "demo"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()

	if err := run([]string{"create", "-d", "first slice", "First task"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	created := out.String()
	if !strings.Contains(created, "First task") {
		t.Fatalf("create output = %q, want title", created)
	}
	fields := strings.Split(strings.TrimSpace(created), "\t")
	if len(fields) < 5 {
		t.Fatalf("create output = %q, want tab fields", created)
	}
	id := fields[0]
	out.Reset()

	if err := run([]string{"label", "add", id, "core", "unix"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	out.Reset()

	if err := run([]string{"update", "--add-label", "git", id}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	out.Reset()

	if err := run([]string{"show", id}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	show := out.String()
	if !strings.Contains(show, "labels\tcore,git,unix") {
		t.Fatalf("show output = %q, want sorted labels", show)
	}
	out.Reset()

	if err := run([]string{"close", "--reason", "done", id}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	out.Reset()

	if err := run([]string{"list"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("list output = %q, want no open issues", out.String())
	}
	out.Reset()

	if err := run([]string{"list", "--all", "--json"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\"status\":\"closed\"") {
		t.Fatalf("json list = %q, want closed bead", out.String())
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runCmd(t, repo, "git", "init")
	runCmd(t, repo, "git", "config", "user.name", "Test User")
	runCmd(t, repo, "git", "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, repo, "git", "add", "README.md")
	runCmd(t, repo, "git", "commit", "-m", "initial")
	return repo
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
