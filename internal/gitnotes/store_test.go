package gitnotes

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	repo := initGitRepo(t)
	store, err := Open(repo)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := store.Init("demo", false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Prefix != "demo" {
		t.Fatalf("prefix = %q, want demo", cfg.Prefix)
	}

	first, err := store.Create(CreateInput{
		Title:       "First task",
		Description: "Ship the first slice",
		Type:        "task",
		Priority:    1,
		Labels:      []string{"core", "unix"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(CreateInput{
		Title:        "Second task",
		Description:  "Waits for the first",
		Type:         "task",
		Dependencies: []string{first.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "First task" {
		t.Fatalf("title = %q, want %q", got.Title, "First task")
	}

	list, err := store.List(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	blocked, err := store.Blocked()
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked) != 1 || blocked[0].ID != second.ID {
		t.Fatalf("blocked = %#v, want only %s", blocked, second.ID)
	}

	if _, err := store.Close(first.ID, "done"); err != nil {
		t.Fatal(err)
	}

	ready, err := store.Ready()
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 || ready[0].ID != second.ID {
		t.Fatalf("ready = %#v, want only %s", ready, second.ID)
	}

	found, err := store.Search("waits", Filter{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != second.ID {
		t.Fatalf("search = %#v, want only %s", found, second.ID)
	}
}

func TestLabelAndDependencyEditing(t *testing.T) {
	repo := initGitRepo(t)
	store, err := Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Init("demo", false); err != nil {
		t.Fatal(err)
	}

	a, err := store.Create(CreateInput{Title: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(CreateInput{Title: "B"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddLabels(a.ID, []string{"alpha", "beta"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RemoveLabels(a.ID, []string{"beta"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddDependency(a.ID, b.ID); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Labels) != 1 || got.Labels[0] != "alpha" {
		t.Fatalf("labels = %#v, want [alpha]", got.Labels)
	}
	if len(got.Dependencies) != 1 || got.Dependencies[0] != b.ID {
		t.Fatalf("dependencies = %#v, want [%s]", got.Dependencies, b.ID)
	}

	if _, err := store.RemoveDependency(a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Dependencies) != 0 {
		t.Fatalf("dependencies = %#v, want empty", got.Dependencies)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	run(t, repo, "git", "init")
	run(t, repo, "git", "config", "user.name", "Test User")
	run(t, repo, "git", "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "git", "add", "README.md")
	run(t, repo, "git", "commit", "-m", "initial")
	return repo
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
