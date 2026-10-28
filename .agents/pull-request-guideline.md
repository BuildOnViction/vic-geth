# Pull Request Guideline

## Guideline

- Keep one logical change per pull request; split unrelated work into separate ones.
- Title the pull request with a short imperative summary of the change (e.g. `Add retry to the payment client`).
- Follow the repository's pull request template when it has one; otherwise write the description in these parts, writing "None" for an empty one:
  - Motivation: the problem or request, with linked issues/tickets.
  - Changes: what changed, grouped by intent.
  - Verification: the checks that ran, with their results; the checks that did not run, and why.
  - Breaking changes: every broken contract, and what its users must do.
  - Follow-ups: findings reported but not fixed.
<!-- Project-specific / Guideline -->

## Checklist

Before opening or updating a pull request, verify:

- [ ] Development checklist is completed (see [development-guideline.md's Checklist](development-guideline.md#checklist)).
- [ ] No failing tests were weakened, skipped, or deleted (see [Testing](development-guideline.md#testing)).
- [ ] Commit messages are clear and describe the *why*, not just the *what*.
- [ ] PR is scoped to a single logical change.
- [ ] PR description explains the motivation and summarizes the change.
- [ ] Breaking changes are called out explicitly.
- [ ] Code duplication is justified.
- [ ] Any new dependency is justified.
- [ ] CI (if configured) passes on the PR branch.
- [ ] Linked issues/tickets are referenced, if applicable.
<!-- Project-specific / Checklist -->

## Project-specific

<!-- Project-specific -->
