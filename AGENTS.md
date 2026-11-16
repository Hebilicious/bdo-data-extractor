# Agent instructions

## Worktrees and pull requests

- Never work directly on `main`. Each task gets its own branch and its own worktree in
  `.worktrees/<branch>`: `git fetch origin && git worktree add .worktrees/<branch> -b <branch> origin/main`.
  Do all the work in that worktree, and keep the main checkout as you found it.
- This repository is a fork. Push the branch to `origin` and open the pull request against this fork's
  `main` (`gh pr create --repo Hebilicious/bdo-data-extractor --base main`), never against the upstream.
- When the pull request merges or closes, remove its worktree (`git worktree remove .worktrees/<branch>`),
  every other worktree you made for the task (a baseline, a comparison).
