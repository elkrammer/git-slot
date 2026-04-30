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
  git-slot list                      list worktrees
  git-slot new <branch>              create local branch and worktree
  git-slot pull <remote/branch>      fetch remote branch and create worktree
`)
}

func repoNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	return name
}

// handles the clone command
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

// handles the new command
func New(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: git-slot new <branch>")
	}

	branch := args[0]

	r, err := ResolveRepo()
	if err != nil {
		return err
	}

	// check if branch already exists locally
	_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	branchExists := err == nil

	worktreePath := r.WorktreePath(branch)
	if branchExists {
		_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "worktree", "add", worktreePath, branch)
	} else {
		_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "worktree", "add", worktreePath, "-b", branch)
	}
	if err != nil {
		return fmt.Errorf("create worktree failed: %w", err)
	}

	fmt.Println(worktreePath)
	return nil
}

// handles the pull command
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

// handles the list command
func List() error {
	r, err := ResolveRepo()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	listOut, err := RunGit(r.GitDir, "--git-dir="+r.GitDir, "worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("list worktrees failed: %w", err)
	}

	worktrees := parseWorktreeList(listOut)

	var active []worktreeInfo
	for _, wt := range worktrees {
		if wt.isBare {
			continue
		}
		wt.isCurrent = wt.path == cwd
		wt.status = getWorktreeStatus(wt.path, wt.branch)
		active = append(active, wt)
	}

	renderWorktreeList(active)
	return nil
}

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	uline  = "\033[4m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[93m"
	cyan   = "\033[96m"
)

func renderWorktreeList(worktrees []worktreeInfo) {
	if len(worktrees) == 0 {
		fmt.Println("No worktrees found.")
		return
	}

	branchPad := 0
	for _, wt := range worktrees {
		if l := len(wt.branch); l > branchPad {
			branchPad = l
		}
	}

	fmt.Printf("%sBranches (%d)%s\n", bold, len(worktrees), reset)

	for _, wt := range worktrees {
		prefix := "  "
		if wt.isCurrent {
			prefix = "* "
		}

		var branch string
		if wt.isCurrent {
			branch = green + bold + fmt.Sprintf("%s%-*s", prefix, branchPad, wt.branch) + reset
		} else {
			branch = fmt.Sprintf("%s%-*s", prefix, branchPad, wt.branch)
		}

		var status string
		switch {
		case wt.status == "clean":
			status = green + "clean" + reset
		case wt.status == "dirty":
			status = yellow + "dirty" + reset
		case strings.Contains(wt.status, "ahead"):
			status = cyan + wt.status + reset
		case strings.Contains(wt.status, "behind"):
			status = red + wt.status + reset
		default:
			status = reset + wt.status + reset
		}

		fmt.Printf("  %s  %s\n", branch, status)
	}
}
