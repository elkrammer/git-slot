package main

type RepoInfo struct {
	Root       string // absolute path to project/ (the directory containing .git/)
	GitDir     string // absolute path to .git/
	IsBare     bool
	CurrentDir string // absolute path to cwd
}

func ResolveRepo() (*RepoInfo, error)                 { return nil, nil }
func (r *RepoInfo) WorktreePath(branch string) string { return "/" }
