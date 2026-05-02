package main

import (
	"fmt"
)

func Completion(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: git-slot completion <shell>")
	}

	shell := args[0]
	switch shell {
	case "zsh":
		fmt.Print(zshCompletion)
	case "bash":
		fmt.Print(bashCompletion)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: zsh, bash)", shell)
	}
	return nil
}

const zshCompletion = `#compdef git-slot

_git_slot_find_gitdir() {
  if [[ -f .git ]]; then
    local gitdir
    gitdir=$(awk '/^gitdir:/ {print $2}' .git 2>/dev/null)
    if [[ -n "$gitdir" ]]; then
      echo "$(cd "$(dirname "$gitdir")/.." && pwd)"
      return
    fi
  fi

  local dir
  for dir in *.git(/); do
    if [[ -f "$dir/config" ]] && grep -q 'bare = true' "$dir/config" 2>/dev/null; then
      echo "$(cd "$dir" && pwd)"
      return
    fi
  done

  local cwd=$PWD
  while [[ "$cwd" != "/" ]]; do
    if [[ -f "$cwd/.git" ]]; then
      local gitdir
      gitdir=$(awk '/^gitdir:/ {print $2}' "$cwd/.git" 2>/dev/null)
      if [[ -n "$gitdir" ]]; then
        echo "$(cd "$(dirname "$gitdir")/.." && pwd)"
        return
      fi
    fi
    for dir in "$cwd"/*.git(/); do
      if [[ -f "$dir/config" ]] && grep -q 'bare = true' "$dir/config" 2>/dev/null; then
        echo "$(cd "$dir" && pwd)"
        return
      fi
    done
    cwd=$(dirname "$cwd")
  done

  return 1
}

_git-slot() {
  local context curcontext="$curcontext" state line
  typeset -A opt_args

  _arguments -C \
    '1:command:(clone completion list ls new pull remove rm)' \
    '2:argument:->argument'

  case $state in
    argument)
      case $words[2] in
        pull)
          local -a branches
          local gitdir
          gitdir=$(_git_slot_find_gitdir)
          if [[ -n "$gitdir" ]]; then
            branches=(${(f)"$(git --git-dir="$gitdir" ls-remote --heads origin 2>/dev/null | while read hash ref; do ref=${ref#refs/heads/}; echo $ref; done)"})
          else
            branches=(${(f)"$(git ls-remote --heads origin 2>/dev/null | while read hash ref; do ref=${ref#refs/heads/}; echo $ref; done)"})
          fi
          if [[ ${#branches} -gt 0 ]]; then
            _describe 'remote branches' branches
          fi
          ;;
        new)
          local -a local_branches
          local gitdir
          gitdir=$(_git_slot_find_gitdir)
          if [[ -n "$gitdir" ]]; then
            local_branches=(${(f)"$(git --git-dir="$gitdir" branch --list 2>/dev/null | sed 's/^[* ] //')"})
          else
            local_branches=(${(f)"$(git branch --list 2>/dev/null | sed 's/^[* ] //')"})
          fi
          if [[ ${#local_branches} -gt 0 ]]; then
            _describe 'local branches' local_branches
          fi
          ;;
        remove|rm)
          local -a worktrees
          local gitdir
          gitdir=$(_git_slot_find_gitdir)
          if [[ -n "$gitdir" ]]; then
            worktrees=(${(f)"$(git --git-dir="$gitdir" worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's|branch refs/heads/||')"})
          else
            worktrees=(${(f)"$(git worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's|branch refs/heads/||')"})
          fi
          if [[ ${#worktrees} -gt 0 ]]; then
            _describe 'worktrees' worktrees
          fi
          ;;
        completion)
          _describe 'shells' '(zsh bash)'
          ;;
      esac
      ;;
  esac
}

compdef _git-slot git-slot
`

const bashCompletion = `# bash completion for git-slot

_git-slot() {
  local cur prev words cword
  _init_completion || return

  if [[ ${#words[@]} -eq 2 ]]; then
    COMPREPLY=($(compgen -W "clone completion list ls new pull remove rm" -- "$cur"))
  elif [[ ${#words[@]} -eq 3 ]]; then
    case "${words[2]}" in
      pull)
        local gitdir
        gitdir=$(_git_slot_find_gitdir_bash)
        if [[ -n "$gitdir" ]]; then
          local branches
          branches=$(git --git-dir="$gitdir" ls-remote --heads origin 2>/dev/null | while read hash ref; do ref=${ref#refs/heads/}; echo "$ref"; done)
          COMPREPLY=($(compgen -W "$branches" -- "$cur"))
        fi
        ;;
      new)
        local gitdir
        gitdir=$(_git_slot_find_gitdir_bash)
        if [[ -n "$gitdir" ]]; then
          local local_branches
          local_branches=$(git --git-dir="$gitdir" branch --list 2>/dev/null | sed 's/^[* ] //')
          COMPREPLY=($(compgen -W "$local_branches" -- "$cur"))
        fi
        ;;
      remove|rm)
        local gitdir
        gitdir=$(_git_slot_find_gitdir_bash)
        if [[ -n "$gitdir" ]]; then
          local worktrees
          worktrees=$(git --git-dir="$gitdir" worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's|branch refs/heads/||')
          COMPREPLY=($(compgen -W "$worktrees" -- "$cur"))
        fi
        ;;
      completion)
        COMPREPLY=($(compgen -W "zsh bash" -- "$cur"))
        ;;
    esac
  fi
}

_git_slot_find_gitdir_bash() {
  if [[ -f .git ]]; then
    local gitdir
    gitdir=$(awk '/^gitdir:/ {print $2}' .git 2>/dev/null)
    if [[ -n "$gitdir" ]]; then
      echo "$(cd "$(dirname "$gitdir")/.." && pwd)"
      return
    fi
  fi

  local dir
  for dir in *.git/; do
    if [[ -f "$dir/config" ]] && grep -q 'bare = true' "$dir/config" 2>/dev/null; then
      echo "$(cd "$dir" && pwd)"
      return
    fi
  done

  local cwd=$PWD
  while [[ "$cwd" != "/" ]]; do
    if [[ -f "$cwd/.git" ]]; then
      local gitdir
      gitdir=$(awk '/^gitdir:/ {print $2}' "$cwd/.git" 2>/dev/null)
      if [[ -n "$gitdir" ]]; then
        echo "$(cd "$(dirname "$gitdir")/.." && pwd)"
        return
      fi
    fi
    for dir in "$cwd"/*.git/; do
      if [[ -f "$dir/config" ]] && grep -q 'bare = true' "$dir/config" 2>/dev/null; then
        echo "$(cd "$dir" && pwd)"
        return
      fi
    done
    cwd=$(dirname "$cwd")
  done

  return 1
}

complete -F _git-slot git-slot
`
