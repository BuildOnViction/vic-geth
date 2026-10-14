# Development Guideline

Source code is written for machines but maintained by humans and machines; write code that both can understand.

## Exploring

When exploring for a solution to the task, try to reuse existing code following this priority order and stop at the first one that is satisfied; this step is needed to make the project easy to maintain and understandable in the long run:

- Does it already exist in this codebase? Reuse the helper, util, or pattern that's already here; don't rewrite it.
- Does the standard library already do this? Use it.
- Does a native platform feature cover it? Use it.
- Does an already-installed dependency solve it? Use it.
- Does a closely matching implementation that needs adaptation exist in this codebase? Report it.
- If none of the above applies, writing new code is fine.
<!-- Project-specific / Exploring -->

## Writing

When writing code, follow these rules:

- Explore carefully before writing; make use of what is available in the codebase first; see [Exploring](#exploring).
- Write the minimum that satisfies the task; never cut safety: validation, error handling, security, accessibility; see [Security](#security).
- Make targeted edits, never whole-file regeneration; every changed line should trace to the request.
- Keep one job per unit at every scale (function, type, package); if describing its job needs "and", split it. Parsing, domain rules, persistence, external calls, presentation, and wiring stay in separate homes.
- Use the narrowest access modifier; keep the public API surface small.
- Grep every caller of the function/method you touch to make sure a signature or behavior change doesn't introduce a new bug or leave a bug half-fixed.
- Report unrelated findings; never do drive-by renames, reformats, or dependency bumps.
- Prefer small, focused functions and files; avoid large "god" packages or files.
- Prefer surrounding code's local, idiomatic style over general rules unless it is unsafe or broken.
- Prefer early exits for invalid or terminal cases over nested conditionals.
- Prefer allowlists over denylists.
- Remove what your change orphaned (unused code, imports, tests, files).
- When assigning values to object fields, follow the order of field declarations if possible.
- When renaming types, methods, functions, remember to check relevant tests.
- When moving files, use the source control move command to retain history.
- Avoid magic numbers or strings; use a named constant for the meaning, or attach an inline *why* comment.
- Avoid sibling variants (`_v2`, `_new`, `_final`, `_copy`); replace the original.
- Avoid unrelated code in `utils`/`helpers`/`common`; name the domain concept instead.
- Avoid unnecessary abstraction; don't introduce an interface until there are two or more real implementations.
- Never modify arguments of functions/methods as side effects; return values instead.
<!-- Project-specific / Writing -->

## Security

When writing code or verifying changes, follow these supplementary rules:

- Use parameterized queries and safe APIs.
- Keep authorization checks beside the operation they protect, or centralized in one enforced policy layer.
- Perform input validation at trust boundaries. Boundaries are external APIs, databases, file systems, clocks, queues, UI events, network calls, subprocesses, generated code.
- Prefer well-maintained standard libraries for crypto, parsing, auth, and serialization.
- Never log secrets, tokens, and sensitive data in error messages and logs.
- Never hardcode secrets or commit them to the repository; read them from the environment or a secret store.
<!-- Project-specific / Security -->

## Fixing

When fixing issues, instructions of [Writing](#writing) apply, plus:

- Fix the root cause, not just the symptom.
- Write a test that reproduces the bug first, watch it fail, then make it pass.
- For refactoring, run the relevant tests before and after; behavior must not change.
- For deduplication, only merge true duplication (copies that must always change together); merging accidental lookalikes is harder to undo than leaving them apart.
<!-- Project-specific / Fixing -->

## Testing

When writing tests, follow these rules:

- Write one concept per test; split a test whose name needs "and"; prefer behavior-focused names.
- Write at least three deterministic expectations, and make sure at least one fails on the untouched fixture.
- Keep the three parts of a test visibly distinct: arrange, act, assert; this is about code structure, don't place comments to distinguish these parts.
- Keep tests F.I.R.S.T.: fast, independent, repeatable, self-validating, timely.
- Keep results deterministic: no sleeps or timing guesses, no real network or filesystem call.
- Assert on outcomes, not implementation.
- Never weaken, skip, or delete failing tests: restore the test, fix the code, or report the conflict and stop.
<!-- Project-specific / Testing -->
- Before running tests that use the fixtures in `tests/testdata`, update the git submodule to the revision pinned in the commit tree (not in a file): `git submodule update --init --checkout tests/testdata`.
- Package rlp/rlpgen must be tested using environment variable `GOTOOLCHAIN=local`.

## Common Mistakes

Check your diff against these failure patterns; each must be fixed before completion:

- Hallucinated API: a function, method, option, or config key not in this codebase or the installed version => verify at the installed version, replace or remove.
- Unverified dependency: a new package or import without checking its registry name, version, or whether an installed dependency does the job => confirm the exact name/version (an invented name may be attacker-registered); prefer what is installed.
- Context loss: a stale or partial read that contradicts existing code, restores deleted code, or drops error handling in a rewrite => re-read target files, callers, tests.
- Scope creep: changed lines that trace to no request such as drive-by renames, reformatting, dependency bumps => revert them and report instead.
- Duplicate implementation: a helper or sibling variant paralleling one that exists => search before writing, extend the original, merge only true duplication, never code owned by different actors.
- Wrong-file gravity: logic added to whatever file was open, a god file growing, a new file at the repository root or current directory => place by role per the project layout, see [project-structure.md](project-structure.md).
- Phantom success: "should work now" with nothing run; a stub, pass, placeholder, or demo value presented as done => run the check and quote the result, name what did not run, and state plainly what remains.
- Test weakening: an assertion loosened; a test skipped, deleted, or rewritten to match the bug, a snapshot re-accepted unread => restore the test and fix the code, or report the conflict and stop.
- Silent architecture drift: an outward import; an ORM, framework, or HTTP type in a business rule; a skipped layer; a new cycle; structure named after the stack => keep imports inward, keep framework types at the boundary, name structure after the domain; when the `Layer Dependencies` table in `.attunic/context.md` declares layers, check every changed file's imports against it.
- Missing validation: an input from a trust boundary accepted unchecked, including empty, null, or boundary values => validate at the boundary where the data enters.
- Missing error handling: a failing path swallowed or ignored, such as an unchecked error or an empty catch => handle it, wrap it, or propagate it explicitly.
<!-- Project-specific / Common Mistakes -->

## Checklist

Before considering any coding task complete, please verify the following to ensure nothing is missed during implementation:

- [ ] The task's requirements are fulfilled; acceptance criteria (if any) are verified against real behavior, not assumed from passing tests.
- [ ] Whole project is built successfully (see [context.md](../.attunic/context.md)).
- [ ] Code files are formatted and pass static analysis (see [context.md](../.attunic/context.md)).
- [ ] Changed code files follow project convention (see [coding-conventions.md](coding-conventions.md)).
- [ ] Declared layers in [context.md](../.attunic/context.md)'s `Layer Dependencies` table (if any) are respected.
- [ ] Changes are scoped to the task, unrelated fixes are reported, not bundled in.
- [ ] The complete final diff has been read line by line; every change is intentional and understood.
- [ ] Changes are reviewed for security requirements (see [Security](#security)).
- [ ] Changes are reviewed for common mistakes (see [Common Mistakes](#common-mistakes)).
- [ ] Every caller of changed functions/methods is checked; no signature or behavior change left half-applied (see [Writing](#writing)).
- [ ] New/changed behavior has test coverage.
- [ ] Relevant tests are passing, except the ones documented as known failures.
- [ ] No secrets, tokens, or sensitive payloads in logs, error messages, or hardcoded values.
- [ ] No unused code, debug prints, or commented-out blocks left behind.
- [ ] New or moved top-level files and folders are reflected in [context.md](../.attunic/context.md)'s `Layout` (prompt the user per its rule).
- [ ] Lasting choices and confirmed facts are recorded per [attunic-protocol.md](attunic-protocol.md)'s Write Rules; ensure nothing is missed: decisions and open questions go to [memory.md](../.attunic/memory.md), confirmed facts to [context.md](../.attunic/context.md)'s `Confirmed Facts`.
- [ ] Every applicable item above was verified by actually running the check; anything not run is explicitly reported, not claimed.
<!-- Project-specific / Checklist -->

## Project-specific

<!-- Project-specific -->
