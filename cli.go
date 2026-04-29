package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Usage() {
	fmt.Println(`git-slot - makes git worktrees behave like a normal directory tree

usage:
  git-slot clone <url> [directory]   clone bare repo and create default worktree
  git-slot pull <remote/branch>      fetch remote branch and create worktree
`)
}

func repoNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	return name
}

func Clone(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: git-slot clone <url> [directory]")
	}

	url := args[0]
	repoName := repoNameFromURL(url)

	targetDir := repoName
	if len(args) >= 2 {
		targetDir = args[1]
	}

	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	if err := os.MkdirAll(absTarget, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	gitDirName := repoName + ".git"
	gitDir := filepath.Join(absTarget, gitDirName)

	_, err = RunGit(absTarget, "clone", "--bare", url, gitDirName)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	defaultBranch, err := RunGit(absTarget, "--git-dir="+gitDir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("detect default branch: %w", err)
	}

	worktreePath := filepath.Join(absTarget, flattenBranch(defaultBranch))
	_, err = RunGit(absTarget, "--git-dir="+gitDir, "worktree", "add", worktreePath, defaultBranch)
	if err != nil {
		return fmt.Errorf("create worktree failed: %w", err)
	}

	fmt.Println(worktreePath)
	return nil
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
