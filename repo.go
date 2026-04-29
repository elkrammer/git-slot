package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RepoInfo struct {
	Root       string // absolute path to project/ (the directory containing .git/)
	GitDir     string // absolute path to .git/
	IsBare     bool
	CurrentDir string // absolute path to cwd
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

		// scan for <name>.git/ directories that are bare
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasSuffix(entry.Name(), ".git") {
					candidate := filepath.Join(dir, entry.Name())
					isBare, err := isBareRepo(candidate)
					if err != nil {
						return nil, err
					}
					if isBare {
						return &RepoInfo{
							Root:       dir,
							GitDir:     candidate,
							IsBare:     true,
							CurrentDir: cwd,
						}, nil
					}
				}
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
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, nil // assume not bare if no config
	}
	return strings.Contains(string(data), "bare = true"), nil
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
