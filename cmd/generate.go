package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/svetozarm/scry/internal/config"
	"github.com/svetozarm/scry/internal/git"
	"github.com/svetozarm/scry/internal/prompt"
	"github.com/svetozarm/scry/internal/provider"
	"github.com/svetozarm/scry/internal/ui"
)

// prompter is the UI abstraction used by the interactive loop.
// Override in tests with a mock.
var prompter ui.Prompter = ui.DefaultPrompter{}

func runGenerate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return handleError(err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return handleError(err)
	}

	cfg, err := config.Load(configPath, cwd, homeDir)
	if err != nil {
		return handleError(err)
	}

	if err := git.EnsureRepo(cwd); err != nil {
		return handleError(err)
	}
	if err := git.EnsureStagedChanges(cwd); err != nil {
		return handleError(err)
	}

	diff, err := git.Diff(cwd)
	if err != nil {
		return handleError(err)
	}
	branch, _ := git.BranchName(cwd)
	author, _ := git.Author(cwd)

	p, err := provider.New(cfg.Provider, cfg.ProviderConfig)
	if err != nil {
		return handleError(err)
	}

	vars := prompt.Vars{BranchName: branch, Author: author}
	maxTokens := p.MaxTokens(cfg.ModelID)

	// If diff exceeds the summary threshold, summarize per-file first.
	useSummary := cfg.DiffSummaryThreshold > 0 && len(diff) > cfg.DiffSummaryThreshold

	var payload string
	var truncated bool

	if useSummary {
		summaries, err := summarizeFiles(cmd.Context(), cwd, p, cfg, nonInteractive)
		if err != nil {
			return handleError(err)
		}
		payload, truncated = prompt.BuildFromSummaries(cfg.Prompt, summaries, vars, maxTokens)
	} else {
		payload, truncated = prompt.Build(cfg.Prompt, diff, vars, maxTokens)
	}

	if truncated {
		if nonInteractive {
			fmt.Fprintln(os.Stderr, "Warning: diff was truncated to fit context window")
		} else {
			ui.DisplayWarning("diff was truncated to fit context window")
		}
	}

	if nonInteractive {
		msg, err := p.Invoke(cmd.Context(), cfg.ModelID, payload)
		if err != nil {
			return handleError(err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), msg)
		return nil
	}

	for {
		msg, err := prompter.WithSpinner("Generating commit message...", func() (string, error) {
			return p.Invoke(cmd.Context(), cfg.ModelID, payload)
		})
		if err != nil {
			return handleError(err)
		}

		prompter.DisplayMessage(msg)

		action, err := prompter.PromptAction()
		if err != nil {
			return handleError(err)
		}

		switch action {
		case ui.ActionAccept:
			output, err := git.Commit(cwd, msg)
			if err != nil {
				return handleError(err)
			}
			prompter.DisplayCommitResult(output)
			return nil
		case ui.ActionRegenerate:
			continue
		case ui.ActionCancel:
			return nil
		}
	}
}

// summarizeFiles gets per-file diffs and summarizes each one via the LLM
// using a worker pool of cfg.SummaryConcurrency goroutines.
func summarizeFiles(ctx context.Context, cwd string, p provider.Provider, cfg *config.Config, quiet bool) (map[string]string, error) {
	files, err := git.DiffFileNames(cwd)
	if err != nil {
		return nil, err
	}

	total := len(files)
	concurrency := cfg.SummaryConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	if !quiet {
		ui.DisplayWarning(fmt.Sprintf("Large diff detected (%d files). Summarizing per-file with %d workers...", total, concurrency))
	} else {
		fmt.Fprintf(os.Stderr, "Warning: large diff detected (%d files), summarizing per-file with %d workers\n", total, concurrency)
	}

	type result struct {
		file    string
		summary string
		err     error
	}

	var (
		mu        sync.Mutex
		completed int
	)

	progress := func(file string) {
		mu.Lock()
		completed++
		n := completed
		mu.Unlock()
		if !quiet {
			ui.DisplayProgressBar(n, total, file)
		} else {
			fmt.Fprintf(os.Stderr, "[%d/%d] Done: %s\n", n, total, file)
		}
	}

	results := make([]result, total)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, file := range files {
		wg.Add(1)
		go func(idx int, f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fileDiff, err := git.DiffFile(cwd, f)
			if err != nil {
				results[idx] = result{file: f, err: err}
				return
			}

			summaryPrompt := prompt.SummaryPrompt(f, fileDiff)
			summary, err := p.Invoke(ctx, cfg.ModelID, summaryPrompt)
			if err != nil {
				results[idx] = result{file: f, err: err}
				return
			}

			progress(f)
			results[idx] = result{file: f, summary: summary}
		}(i, file)
	}

	wg.Wait()

	if !quiet {
		fmt.Fprint(os.Stderr, "\n")
	}

	summaries := make(map[string]string, total)
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		summaries[r.file] = r.summary
	}
	return summaries, nil
}
