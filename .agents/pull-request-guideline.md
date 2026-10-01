# Pull Request Guideline

## Checklist

Before opening or updating a pull request, verify:

- [ ] Whole project is built successfully (see [commands.md](commands.md)).
- [ ] Code files are formatted, passed static analysis (see [commands.md](commands.md)).
- [ ] Imports are tidy (see [commands.md](commands.md)).
- [ ] All tests are passed, except the ones documented as known failures (see [commands.md](commands.md)).
- [ ] Commit messages are clear and describe the *why*, not just the *what*.
- [ ] PR description explains the motivation and summarizes the change.
- [ ] PR is scoped to a single logical change - split unrelated changes into separate PRs.
- [ ] Breaking changes are called out explicitly in the PR description.
- [ ] Code duplication is justified (the same logic should not be duplicated, as it hurts maintainability).
- [ ] Any new dependency is justified (avoid adding dependencies for trivial functionality).
- [ ] CI (if configured) passes on the PR branch.
- [ ] Linked issues/tickets are referenced, if applicable.
- [ ] Working tree contains no conflict markers and no `*.rej` files (see [task-guideline.md](task-guideline.md)).
<!-- Project-specific / Checklist -->

## Project-specific

<!-- Project-specific -->
