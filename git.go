package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// runs git command in the given directory and returns stdout
func RunGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func parseWorktreeList(output string) []worktreeInfo {
	var result []worktreeInfo
	var current *worktreeInfo

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				result = append(result, *current)
				current = nil
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current = &worktreeInfo{path: strings.TrimPrefix(line, "worktree ")}
		} else if current != nil {
			switch {
			case strings.HasPrefix(line, "HEAD "):
				// ignore
			case strings.HasPrefix(line, "branch "):
				current.branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
			case line == "bare":
				current.isBare = true
			}
		}
	}
	if current != nil {
		result = append(result, *current)
	}
	return result
}

func getWorktreeStatus(path, branch string) string {
	statusOut, err := RunGit(path, "-C", path, "status", "--porcelain")
	if err == nil && strings.TrimSpace(statusOut) != "" {
		return "dirty"
	}

	trackOut, err := RunGit(path, "-C", path, "for-each-ref", "--format=%(upstream:track)", "refs/heads/"+branch)
	if err == nil {
		track := strings.TrimSpace(trackOut)
		if track != "" {
			return track
		}
	}

	return "clean"
}
