# git-slot

git-slot is an opinionated tool that turns Git worktrees into a directory-based workflow.

No more `git stash` gymnastics. no more "oh shit i committed to the wrong branch".
Every branch gets its own directory you can switch between instantly.
Git worktrees are powerful, but the default workflow is fragmented and awkward. git-slot removes the cognitive load.

## the problem

Here's what Git worktrees look like in practice.
1. you clone a repo:

```
~/code/
  my-project/         # your main clone, branch = main
```

you want to work on a feature, so you create a worktree:

```bash
cd ~/code/my-project
git worktree add ../my-project-feature feature-auth
```

now your code lives in two completely different places:

```
~/code/
  my-project/              # branch: main
  my-project-feature/      # branch: feature-auth
```

then you make another one for a hotfix:

```bash
git worktree add ~/tmp/hotfix-thing hotfix-123
```

now you have this mess:

```
~/code/
  my-project/
  my-project-feature/
~/tmp/
  hotfix-thing/
```

the problem isn't Git worktrees themselves. the problem is that branches and directories are completely decoupled.
your branch is called `feature-auth` but it lives in `../my-project-feature`.
your branch is called `hotfix-123` but it lives in `~/tmp/hotfix-thing`.

every worktree becomes another thing you have to remember:

- where did i put it?
- what did i name the directory?
- how do i get back to it?
- what was the exact git worktree add syntax again?

that's cognitive load i don't want.

## the idea

git-slot organizes worktrees like this instead:

```
project/
  project.git/      # bare repo
  main/             # branch: main
  feature-auth/     # branch: feature-auth
  hotfix-123/       # branch: hotfix-123
```

one project directory. one directory per branch. the folder name is the branch name.
flat. predictable. no path juggling.
stand in `project/` and see all your branches instantly.

`cd feature-auth` and you're in that branch.
your shell, editor, file manager, fuzzy finder, and terminal already understand directories.
git-slot leans into that instead of inventing another abstraction.

## philosophy

branches should behave like directories
one project root, everything underneath it
no hidden state
no path guessing
no config
leverage the filesystem instead of fighting it

## install

```bash
go install github.com/elkrammer/git-slot@latest
```

or build manually:

```bash
go build -o git-slot
```

## usage

```bash
# clone a repo and initialize it with git-slot layout
git-slot clone https://github.com/user/repo.git
cd repo
git-slot new feature-auth        # create a new local branch + worktree
git-slot pull origin/some-branch # fetch remote branch + create worktree
git-slot list                    # list checked out branches / worktrees
```


## shell integration

add this to your `.bashrc` / `.zshrc` to enable completion

```bash
eval "$(git-slot completion bash)"
# or
eval "$(git-slot completion zsh)"
```


## notes

- branch names containing / are sanitized to -
- feature/auth -> feature-auth
- remote tracking branches are created automatically when using pull
- git-slot uses standard Git worktrees underneath
- zero config

## license

MIT
