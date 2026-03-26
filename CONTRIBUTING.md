# Contributing

Thanks for your interest in contributing.

## Quick Rules

- Be respectful and constructive.
- Keep pull requests focused and reviewable.
- Humans own final code, tests, docs, and commits.
- Disclose meaningful AI assistance in PR descriptions.

## Reporting Issues

Use the bug report template. Include: Go version, OS, reproduction steps, and the output.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add retry on 5xx errors
fix: skip URLs inside fenced code blocks
docs: add skip-pattern usage examples
refactor: extract URL dedup into helper
chore: upgrade to Go 1.23
```

Types: `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `chore`, `ci`

Breaking changes: add `!` after the type (`feat!: rename flag`) — this signals a MAJOR version bump.

## Pull Requests

- Open an issue first for non-trivial changes.
- Keep PRs small and scoped to one change.
- Update `CHANGELOG.md` if adding a feature or fixing a bug.
- Ensure `go build ./...` and `go vet ./...` pass.

## AI Contribution Policy

AI-assisted work is allowed, but requires human accountability.

Required in each PR:
- AI tools/models used (if any)
- Files or sections materially influenced by AI
- Human validation performed (build, test, review)

Prohibited in commit history:
- AI branding lines and AI co-author trailers

Install the pre-commit hook to catch issues before push:

```sh
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## Zero Dependencies Policy

This project uses the Go standard library only. PRs that add external dependencies will not be accepted unless there is a compelling reason discussed in an issue first.
