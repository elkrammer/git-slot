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

// walks up from cwd looking for a bare .git directory.
func ResolveRepo() (*RepoInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	dir := cwd
	for {
		gitDir := filepath.Join(dir, ".git")
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			// found .git/, check if bare
			isBare, err := isBareRepo(gitDir)
			if err != nil {
				return nil, err
			}
			if isBare {
				return &RepoInfo{
					Root:       dir,
					GitDir:     gitDir,
					IsBare:     true,
					CurrentDir: cwd,
				}, nil
			}
			// .git exists but not bare; could be a regular clone or worktree; check if it's a worktree gitdir file
			if isWorktreeGitdir(gitDir) {
				bareDir, err := readWorktreeGitdir(gitDir)
				if err != nil {
					return nil, err
				}
				root := filepath.Dir(bareDir)
				return &RepoInfo{
					Root:       root,
					GitDir:     bareDir,
					IsBare:     true,
					CurrentDir: cwd,
				}, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, fmt.Errorf("not in a git-slot repository (no bare .git found)")
}

func (r *RepoInfo) WorktreePath(branch string) string { return "/" }

func isBareRepo(gitDir string) (bool, error) {
	configPath := filepath.Join(gitDir, "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, nil // assume not bare if no config
	}
	return strings.Contains(string(data), "bare = true"), nil
}

func isWorktreeGitdir(gitDir string) bool {
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	// if .git is a file, it's a worktree gitdir link
	return !info.IsDir()
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
