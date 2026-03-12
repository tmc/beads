//go:build scripttests
// +build scripttests

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("scripttest uses Unix shell commands (sh -c), skipping on Windows")
	}

	exeName := "bd"
	binDir := t.TempDir()
	exe := filepath.Join(binDir, exeName)
	if err := exec.Command("go", "build", "-o", exe, ".").Run(); err != nil {
		t.Fatal(err)
	}

	timeout := 2 * time.Second
	cmds := scripttest.DefaultCmds()
	idRe := regexp.MustCompile(`\b([a-zA-Z]+-[a-z0-9]{3,}(?:\.[0-9]+)?)\b`)

	cmds["bd"] = script.Command(
		script.CmdUsage{
			Summary: "run bd and cache the last issue ID in $ID",
			Args:    "args...",
			Async:   true,
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			cmd := exec.CommandContext(s.Context(), exe, args...)
			cmd.Dir = s.Getwd()
			cmd.Env = append([]string(nil), s.Environ()...)

			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Start(); err != nil {
				return nil, err
			}

			return func(s *script.State) (string, string, error) {
				err := cmd.Wait()
				out := stdout.String()
				if recordErr := recordCommandStdout(s, out); recordErr != nil {
					return "", "", recordErr
				}
				if matches := idRe.FindAllStringSubmatch(out, -1); len(matches) > 0 {
					s.Setenv("ID", matches[len(matches)-1][1])
				}
				return out, stderr.String(), err
			}, nil
		},
	)

	cmds["git"] = programWithCapturedStdout("git", timeout)
	cmds["sh"] = programWithCapturedStdout("sh", timeout)

	cmds["jq"] = script.Command(
		script.CmdUsage{
			Summary: "run jq against previous stdout",
			Args:    "filter [file]",
			Async:   true,
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("jq requires a filter")
			}

			cmd := exec.CommandContext(s.Context(), "jq", args...)
			cmd.Dir = s.Getwd()
			if len(args) == 1 {
				input := s.Stdout()
				if inputFile, ok := s.LookupEnv("SCRIPT_JQ_INPUT"); ok && inputFile != "" {
					data, err := os.ReadFile(inputFile)
					if err != nil {
						return nil, fmt.Errorf("read jq input: %w", err)
					}
					input = string(data)
				}
				cmd.Stdin = strings.NewReader(input)
			}

			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Start(); err != nil {
				return nil, err
			}

			return func(*script.State) (string, string, error) {
				err := cmd.Wait()
				return stdout.String(), stderr.String(), err
			}, nil
		},
	)

	cmds["extractid"] = script.Command(
		script.CmdUsage{
			Summary: "extract the Nth issue ID from stdout into an env var",
			Args:    "VARNAME [N]",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("extractid requires VARNAME [N]")
			}

			n := 1
			if len(args) == 2 {
				parsed, err := strconv.Atoi(args[1])
				if err != nil || parsed < 1 {
					return nil, fmt.Errorf("N must be a positive integer")
				}
				n = parsed
			}

			matches := idRe.FindAllStringSubmatch(s.Stdout(), -1)
			if len(matches) < n {
				return nil, fmt.Errorf("only found %d issue IDs in stdout, wanted %d", len(matches), n)
			}

			return nil, s.Setenv(args[0], matches[n-1][1])
		},
	)

	cmds["gitinit"] = script.Command(
		script.CmdUsage{
			Summary: "initialize a git repo with test identity and an initial commit",
			Args:    "[commit message]",
			Async:   true,
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			msg := "Initial commit"
			if len(args) > 0 {
				msg = strings.Join(args, " ")
			}

			cmd := exec.CommandContext(s.Context(), "sh", "-c", fmt.Sprintf(`
				git init &&
				git config user.name 'Test User' &&
				git config user.email 'test@example.com' &&
				git config commit.gpgsign false &&
				printf '# Test Project\n' > README.md &&
				git add README.md &&
				git commit -m %q
			`, msg))
			cmd.Dir = s.Getwd()
			cmd.Env = s.Environ()

			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Start(); err != nil {
				return nil, err
			}

			return func(*script.State) (string, string, error) {
				err := cmd.Wait()
				out := stdout.String()
				if recordErr := recordCommandStdout(s, out); recordErr != nil {
					return "", "", recordErr
				}
				return out, stderr.String(), err
			}, nil
		},
	)

	conds := scripttest.DefaultConds()
	conds["dolt"] = script.BoolCondition(
		"shared Dolt test server is available",
		os.Getenv("BEADS_DOLT_PORT") != "",
	)

	engine := &script.Engine{
		Cmds:  cmds,
		Conds: conds,
	}

	env := append([]string(nil), os.Environ()...)
	env = setEnvVar(env, "PATH", binDir+":"+os.Getenv("PATH"))
	env = setEnvVar(env, "BEADS_TEST_MODE", "1")
	env = setEnvVar(env, "GIT_CONFIG_GLOBAL", "/dev/null")
	env = setEnvVar(env, "GIT_CONFIG_SYSTEM", "/dev/null")

	if os.Getenv("BEADS_DOLT_PORT") == "" {
		t.Log("[dolt] scripts will be skipped: BEADS_DOLT_PORT is not set")
	}

	scripttest.Test(t, context.Background(), engine, env, "testdata/*.txt")
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func programWithCapturedStdout(name string, timeout time.Duration) script.Cmd {
	return script.Command(
		script.CmdUsage{
			Summary: "run " + name + " and preserve stdout for repeated jq filters",
			Args:    "args...",
			Async:   true,
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			cmd := exec.CommandContext(s.Context(), name, args...)
			cmd.Dir = s.Getwd()
			cmd.Env = append([]string(nil), s.Environ()...)

			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Start(); err != nil {
				return nil, err
			}

			return func(s *script.State) (string, string, error) {
				err := cmd.Wait()
				out := stdout.String()
				if recordErr := recordCommandStdout(s, out); recordErr != nil {
					return "", "", recordErr
				}
				return out, stderr.String(), err
			}, nil
		},
	)
}

func recordCommandStdout(s *script.State, out string) error {
	stdoutPath := filepath.Join(s.Getwd(), ".scripttest-last-stdout")
	if err := os.WriteFile(stdoutPath, []byte(out), 0o666); err != nil {
		return fmt.Errorf("write jq input: %w", err)
	}
	return s.Setenv("SCRIPT_JQ_INPUT", stdoutPath)
}
