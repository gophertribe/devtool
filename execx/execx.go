package execx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/gophertribe/devtool/console"
)

// Options configures command execution.
type Options struct {
	Env      map[string]string
	Dir      string
	Stdout   io.Writer
	Stderr   io.Writer
	NoBanner bool
}

// Run executes a command while streaming stdout and stderr.
func Run(name string, args ...string) error {
	return RunWith(Options{}, name, args...)
}

// RunWithV executes a command with environment overrides and prints the full invocation.
func RunWithV(env map[string]string, name string, args ...string) error {
	return RunWith(Options{Env: env}, name, args...)
}

// RunWith executes a command with the provided options.
func RunWith(opts Options, name string, args ...string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("command name is required")
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	if !opts.NoBanner {
		renderBanner(stdout, opts, name, args)
	}

	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	cmd.Env = mergeEnv(opts.Env)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	startedAt := time.Now()
	if err := cmd.Run(); err != nil {
		printFailure(stderr, time.Since(startedAt), err)
		return err
	}

	printSuccess(stdout, time.Since(startedAt))
	return nil
}

func renderBanner(w io.Writer, opts Options, name string, args []string) {
	var parts []string
	if opts.Dir != "" {
		parts = append(parts, "cd "+quoteShellArg(opts.Dir))
	}

	envPairs := sortedEnvPairs(opts.Env)
	if len(envPairs) > 0 {
		parts = append(parts, strings.Join(envPairs, " "))
	}
	parts = append(parts, joinCommand(name, args))

	fmt.Fprintf(w, "%s %s\n", console.Cyan(console.Bold("==>")), console.White(strings.Join(parts, " ")))
}

func printSuccess(w io.Writer, elapsed time.Duration) {
	fmt.Fprintf(w, "%s %s %s\n", console.Green("OK"), console.Dim("completed in"), console.Dim(formatDuration(elapsed)))
}

func printFailure(w io.Writer, elapsed time.Duration, err error) {
	fmt.Fprintf(w, "%s %s %s: %v\n", console.Red("ERR"), console.Dim("failed after"), console.Dim(formatDuration(elapsed)), err)
}

func mergeEnv(overrides map[string]string) []string {
	base := os.Environ()
	if len(overrides) == 0 {
		return base
	}

	values := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))

	for _, entry := range base {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = value
	}

	for key, value := range overrides {
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = value
	}

	env := make([]string, 0, len(order))
	for _, key := range order {
		env = append(env, key+"="+values[key])
	}
	return env
}

func sortedEnvPairs(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+quoteShellArg(values[key]))
	}
	return pairs
}

func joinCommand(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteShellArg(name))
	for _, arg := range args {
		parts = append(parts, quoteShellArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteShellArg(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\n'\"\\$&|;<>()[]{}*?!#~`") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func formatDuration(elapsed time.Duration) string {
	if elapsed < time.Second {
		return elapsed.Round(time.Millisecond).String()
	}
	return elapsed.Round(10 * time.Millisecond).String()
}
