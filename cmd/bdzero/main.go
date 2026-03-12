package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steveyegge/beads/internal/gitnotes"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stdout)
		return nil
	}
	switch args[0] {
	case "help":
		usage(stdout)
		return nil
	case "version":
		fmt.Fprintln(stdout, version)
		return nil
	case "init":
		return runInit(args[1:], stdout)
	case "create":
		return runCreate(args[1:], stdout, false)
	case "q":
		return runCreate(args[1:], stdout, true)
	case "list":
		return runList(args[1:], stdout)
	case "show":
		return runShow(args[1:], stdout)
	case "update":
		return runUpdate(args[1:], stdout)
	case "close":
		return runClose(args[1:], stdout)
	case "reopen":
		return runReopen(args[1:], stdout)
	case "search":
		return runSearch(args[1:], stdout)
	case "ready":
		return runReady(args[1:], stdout)
	case "blocked":
		return runBlocked(args[1:], stdout)
	case "export":
		return runExport(args[1:], stdout)
	case "label":
		return runLabel(args[1:], stdout)
	case "dep":
		return runDep(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `bdzero: zero-dependency beads on git-notes

Usage:
  bdzero init [-p prefix]
  bdzero create [flags] <title>
  bdzero q [flags] <title>
  bdzero list [--all] [--status open|closed] [--type task] [--label name] [--json]
  bdzero show [--json] <id>
  bdzero update [flags] <id>
  bdzero close [--reason text] [--json] <id>
  bdzero reopen [--json] <id>
  bdzero search [flags] <query>
  bdzero ready [--json]
  bdzero blocked [--json]
  bdzero label add|remove|list ...
  bdzero dep add|remove|list ...
  bdzero export [-o file]
  bdzero version

Defaults:
  - Plain text by default.
  - Data lives in refs/notes/beads/*.
  - No hooks are installed.
`)
}

func openStore() (*gitnotes.Store, error) {
	return gitnotes.Open(".")
}

func runInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	prefix := fs.String("p", "", "issue prefix")
	force := fs.Bool("force", false, "overwrite existing config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	if *prefix == "" {
		*prefix = gitnotes.DefaultPrefixFromRepo(store.Repo())
	}
	cfg, err := store.Init(*prefix, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "initialized\tprefix=%s\trepo=%s\n", cfg.Prefix, store.Repo())
	return nil
}

func runCreate(args []string, stdout io.Writer, quiet bool) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	description := fs.String("d", "", "description")
	typ := fs.String("t", "task", "type")
	priority := fs.Int("p", 2, "priority")
	labels := fs.String("l", "", "comma-separated labels")
	deps := fs.String("dep", "", "comma-separated dependency ids")
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("title is required")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	bead, err := store.Create(gitnotes.CreateInput{
		Title:        strings.Join(fs.Args(), " "),
		Description:  *description,
		Type:         *typ,
		Priority:     *priority,
		Labels:       splitCSV(*labels),
		Dependencies: splitCSV(*deps),
	})
	if err != nil {
		return err
	}
	if quiet {
		fmt.Fprintln(stdout, bead.ID)
		return nil
	}
	return printOne(stdout, bead, *jsonOut)
}

func runList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	all := fs.Bool("all", false, "include closed")
	status := fs.String("status", "", "status filter")
	typ := fs.String("type", "", "type filter")
	label := fs.String("label", "", "label filter")
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	list, err := store.List(gitnotes.Filter{
		All:    *all,
		Status: strings.TrimSpace(*status),
		Type:   strings.TrimSpace(*typ),
		Label:  strings.TrimSpace(*label),
	})
	if err != nil {
		return err
	}
	return printList(stdout, list, *jsonOut)
}

func runShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("show requires an id")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	bead, err := store.Get(fs.Arg(0))
	if err != nil {
		return err
	}
	return printOne(stdout, bead, *jsonOut)
}

func runUpdate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	title := fs.String("title", "", "new title")
	description := fs.String("d", "", "new description")
	typ := fs.String("t", "", "new type")
	priority := fs.Int("p", -1, "new priority")
	labels := fs.String("labels", "", "replace labels")
	addLabel := fs.String("add-label", "", "labels to add")
	removeLabel := fs.String("remove-label", "", "labels to remove")
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("update requires an id")
	}
	seen := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		seen[f.Name] = true
	})
	var in gitnotes.UpdateInput
	if seen["title"] {
		in.Title = title
	}
	if seen["d"] {
		in.Description = description
	}
	if seen["t"] {
		in.Type = typ
	}
	if seen["p"] {
		in.Priority = priority
	}
	if seen["labels"] {
		in.Labels = splitCSV(*labels)
	}
	in.AddLabels = splitCSV(*addLabel)
	in.RemoveLabels = splitCSV(*removeLabel)

	store, err := openStore()
	if err != nil {
		return err
	}
	bead, err := store.Update(fs.Arg(0), in)
	if err != nil {
		return err
	}
	return printOne(stdout, bead, *jsonOut)
}

func runClose(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("close", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reason := fs.String("reason", "", "close reason")
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("close requires an id")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	bead, err := store.Close(fs.Arg(0), *reason)
	if err != nil {
		return err
	}
	return printOne(stdout, bead, *jsonOut)
}

func runReopen(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("reopen", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("reopen requires an id")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	bead, err := store.Reopen(fs.Arg(0))
	if err != nil {
		return err
	}
	return printOne(stdout, bead, *jsonOut)
}

func runSearch(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	all := fs.Bool("all", false, "include closed")
	status := fs.String("status", "", "status filter")
	typ := fs.String("type", "", "type filter")
	label := fs.String("label", "", "label filter")
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("search requires a query")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	list, err := store.Search(strings.Join(fs.Args(), " "), gitnotes.Filter{
		All:    *all,
		Status: strings.TrimSpace(*status),
		Type:   strings.TrimSpace(*typ),
		Label:  strings.TrimSpace(*label),
	})
	if err != nil {
		return err
	}
	return printList(stdout, list, *jsonOut)
}

func runReady(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("ready", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	list, err := store.Ready()
	if err != nil {
		return err
	}
	return printList(stdout, list, *jsonOut)
}

func runBlocked(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("blocked", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	list, err := store.Blocked()
	if err != nil {
		return err
	}
	return printList(stdout, list, *jsonOut)
}

func runExport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outFile := fs.String("o", "", "output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	list, err := store.Export()
	if err != nil {
		return err
	}
	var w io.Writer = stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	enc := json.NewEncoder(w)
	for _, bead := range list {
		if err := enc.Encode(bead); err != nil {
			return err
		}
	}
	return nil
}

func runLabel(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("label requires a subcommand")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	switch args[0] {
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("label add requires <id> <labels>")
		}
		bead, err := store.AddLabels(args[1], args[2:])
		if err != nil {
			return err
		}
		return printList(stdout, []*gitnotes.Bead{bead}, false)
	case "remove":
		if len(args) < 3 {
			return fmt.Errorf("label remove requires <id> <labels>")
		}
		bead, err := store.RemoveLabels(args[1], args[2:])
		if err != nil {
			return err
		}
		return printList(stdout, []*gitnotes.Bead{bead}, false)
	case "list":
		if len(args) != 2 {
			return fmt.Errorf("label list requires <id>")
		}
		bead, err := store.Get(args[1])
		if err != nil {
			return err
		}
		for _, label := range bead.Labels {
			fmt.Fprintln(stdout, label)
		}
		return nil
	default:
		return fmt.Errorf("unknown label subcommand %q", args[0])
	}
}

func runDep(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("dep requires a subcommand")
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	switch args[0] {
	case "add":
		if len(args) != 3 {
			return fmt.Errorf("dep add requires <id> <dependency>")
		}
		bead, err := store.AddDependency(args[1], args[2])
		if err != nil {
			return err
		}
		return printList(stdout, []*gitnotes.Bead{bead}, false)
	case "remove":
		if len(args) != 3 {
			return fmt.Errorf("dep remove requires <id> <dependency>")
		}
		bead, err := store.RemoveDependency(args[1], args[2])
		if err != nil {
			return err
		}
		return printList(stdout, []*gitnotes.Bead{bead}, false)
	case "list":
		if len(args) != 2 {
			return fmt.Errorf("dep list requires <id>")
		}
		bead, err := store.Get(args[1])
		if err != nil {
			return err
		}
		for _, dep := range bead.Dependencies {
			fmt.Fprintln(stdout, dep)
		}
		return nil
	default:
		return fmt.Errorf("unknown dep subcommand %q", args[0])
	}
}

func printOne(w io.Writer, bead *gitnotes.Bead, jsonOut bool) error {
	if jsonOut {
		return json.NewEncoder(w).Encode(bead)
	}
	fmt.Fprintf(w, "%s\t%s\tP%d\t%s\t%s\n", bead.ID, bead.Status, bead.Priority, bead.Type, bead.Title)
	if bead.Description != "" {
		fmt.Fprintln(w, bead.Description)
	}
	if len(bead.Labels) > 0 {
		fmt.Fprintf(w, "labels\t%s\n", strings.Join(bead.Labels, ","))
	}
	if len(bead.Dependencies) > 0 {
		fmt.Fprintf(w, "deps\t%s\n", strings.Join(bead.Dependencies, ","))
	}
	if bead.CloseReason != "" {
		fmt.Fprintf(w, "reason\t%s\n", bead.CloseReason)
	}
	return nil
}

func printList(w io.Writer, list []*gitnotes.Bead, jsonOut bool) error {
	if jsonOut {
		return json.NewEncoder(w).Encode(list)
	}
	for _, bead := range list {
		fmt.Fprintf(w, "%s\t%s\tP%d\t%s\t%s\n", bead.ID, bead.Status, bead.Priority, bead.Type, bead.Title)
	}
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func init() {
	flag.CommandLine.SetOutput(io.Discard)
}
