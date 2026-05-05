package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RepoInfo struct {
	Root       string // absolute path to project/ (the directory containing .git/)
	GitDir     string // absolute path to .git/
	IsBare     bool
	CurrentDir string // absolute path to cwd
}

type worktreeInfo struct {
	path      string
	branch    string
	isBare    bool
	isCurrent bool
	status    string
}

// walks up from cwd looking for a bare repo directory
func ResolveRepo() (*RepoInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	dir := cwd
	for {
		gitDot := filepath.Join(dir, ".git")
		info, err := os.Stat(gitDot)
		if err == nil {
			if info.IsDir() {
				// found .git/ directory, check if bare
				isBare, err := isBareRepo(gitDot)
				if err != nil {
					return nil, err
				}
				if isBare {
					return &RepoInfo{
						Root:       dir,
						GitDir:     gitDot,
						IsBare:     true,
						CurrentDir: cwd,
					}, nil
				}
				// .git/ exists but it's not bare  it's a regular repo, does not have our layout
			} else {
				// .git is a file - worktree gitdir link
				bareDir, err := readWorktreeGitdir(gitDot)
				if err == nil {
					root := filepath.Dir(bareDir)
					return &RepoInfo{
						Root:       root,
						GitDir:     bareDir,
						IsBare:     true,
						CurrentDir: cwd,
					}, nil
				}
			}
		}

		// scan for bare repo directories
		entries, err := os.ReadDir(dir)
		if err == nil {
			var bareCandidates []string
			for _, entry := range entries {
				if entry.IsDir() {
					candidate := filepath.Join(dir, entry.Name())
					isBare, err := isBareRepo(candidate)
					if err != nil {
						return nil, err
					}
					if isBare {
						bareCandidates = append(bareCandidates, candidate)
					}
				}
			}

			if len(bareCandidates) == 1 {
				return &RepoInfo{
					Root:       dir,
					GitDir:     bareCandidates[0],
					IsBare:     true,
					CurrentDir: cwd,
				}, nil
			}
			if len(bareCandidates) > 1 {
				// prefer *.git/ directories
				var gitCandidates []string
				for _, c := range bareCandidates {
					if strings.HasSuffix(filepath.Base(c), ".git") {
						gitCandidates = append(gitCandidates, c)
					}
				}
				if len(gitCandidates) == 1 {
					return &RepoInfo{
						Root:       dir,
						GitDir:     gitCandidates[0],
						IsBare:     true,
						CurrentDir: cwd,
					}, nil
				}
				return nil, fmt.Errorf("multiple bare repos found in %s: %v", dir, bareCandidates)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, fmt.Errorf("not in a git-slot repository (no bare repo found)")
}

func flattenBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

func (r *RepoInfo) WorktreePath(branch string) string {
	return filepath.Join(r.Root, flattenBranch(branch))
}

func isBareRepo(gitDir string) (bool, error) {
	configPath := filepath.Join(gitDir, "config")
	cmd := exec.Command("git", "config", "--file", configPath, "--get", "core.bare")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// assume this repo is not bare
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("checking if %s is bare: %w\n%s", gitDir, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func readWorktreeGitdir(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "gitdir: "); ok {
			dir := after
			// bare repo is two levels up
			bareDir := filepath.Dir(filepath.Dir(dir))
			return bareDir, nil
		}
	}
	return "", fmt.Errorf("no gitdir line found in %s", path)
}
