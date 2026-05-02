package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
		os.RemoveAll(absTarget)
		return fmt.Errorf("clone failed: %w", err)
	}

	// set up remote tracking branches so origin/<branch> refs exist
	_, err = RunGit(absTarget, "--git-dir="+gitDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if err != nil {
		os.RemoveAll(absTarget)
		return fmt.Errorf("set fetch config: %w", err)
	}

	defaultBranch, err := RunGit(absTarget, "--git-dir="+gitDir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		os.RemoveAll(absTarget)
		return fmt.Errorf("detect default branch: %w", err)
	}
	// strip refs/heads/ and heads/ prefixes that some git versions return
	defaultBranch = strings.TrimPrefix(defaultBranch, "refs/heads/")
	defaultBranch = strings.TrimPrefix(defaultBranch, "heads/")

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
	r, err := ResolveRepo()
	if err != nil {
		return err
	}

	var remote, branch string

	if len(args) < 1 {
		// fzf mode
		if _, err := exec.LookPath("fzf"); err != nil {
			return fmt.Errorf("fzf not found. install from https://github.com/junegunn/fzf")
		}

		// fetch origin with explicit refspec to populate remote tracking branches
		_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "fetch", "origin", "refs/heads/*:refs/remotes/origin/*")
		if err != nil {
			return fmt.Errorf("fetch origin failed: %w", err)
		}

		// get branch list
		out, err := RunGit(r.GitDir, "--git-dir="+r.GitDir, "ls-remote", "--heads", "origin")
		if err != nil {
			return fmt.Errorf("list branches failed: %w", err)
		}

		// parse branches
		var branches []string
		for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			branchName := strings.TrimPrefix(fields[1], "refs/heads/")
			if branchName != "" {
				branches = append(branches, branchName)
			}
		}

		// run fzf
		cmd := exec.Command("fzf")
		cmd.Stdin = strings.NewReader(strings.Join(branches, "\n"))
		var buf bytes.Buffer
		cmd.Stdout = &buf
		if err := cmd.Run(); err != nil {
			return nil // cancelled
		}
		branch = strings.TrimSpace(buf.String())
		if branch == "" {
			return nil
		}
	} else {
		// explicit branch
		remoteBranch := args[0]
		parts := strings.SplitN(remoteBranch, "/", 2)
		if len(parts) == 2 {
			remote, branch = parts[0], parts[1]
		} else {
			remote, branch = "origin", remoteBranch
		}

		_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "fetch", remote, branch)
		if err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}
	}

	// create worktree
	worktreePath := r.WorktreePath(branch)

	// fetch the specific branch first so origin/<branch> exists as remote tracking ref
	_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "fetch", "origin", "refs/heads/"+branch+":refs/remotes/origin/"+branch)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	branchExists := err == nil

	if branchExists {
		_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "worktree", "add", worktreePath, branch)
	} else {
		_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "worktree", "add", worktreePath, "-b", branch, "origin/"+branch)
	}
	if err != nil {
		return fmt.Errorf("create worktree failed: %w", err)
	}
	// set upstream tracking (worktree add doesn't always do this)
	_, err = RunGit(r.GitDir, "--git-dir="+r.GitDir, "branch", "--set-upstream-to=origin/"+branch, branch)
	if err != nil {
		return fmt.Errorf("set upstream failed: %w", err)
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
