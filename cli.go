package main

import (
	"fmt"
	"strings"
)

func Usage() {
	fmt.Println(`git-slot - makes git worktrees behave like a normal directory tree

usage:
  git-slot pull <remote/branch>    fetch remote branch and create worktree
`)
}

// handles the pull command.
func Pull(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: git-slot pull <remote/branch>")
	}

	remoteBranch := args[0]
	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected remote/branch, got %q", remoteBranch)
	}
	remote, branch := parts[0], parts[1]

	r, err := ResolveRepo()
	if err != nil {
		return err
	}

	// fetch
	_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "fetch", remote, branch)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	// create worktree
	worktreePath := r.WorktreePath(branch)
	_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "worktree", "add", worktreePath, branch)
	if err != nil {
		return fmt.Errorf("create worktree failed: %w", err)
	}

	fmt.Println(worktreePath)
	return nil
}
