# Task Guideline

Source code is written for machines, but is maintained by both humans and machines. Therefore writing good code is important for both human and machine to understand.

## Guideline

Important rules when working with the project; these must be strictly followed:

- Follow instructions: anything that is explicitly requested.
- Planning first: plan by default, unless the user explicitly says to skip it.
- Ask when in doubt: if you are unsure about an instruction, need help, or have a better solution, feel free to ask questions: "Do you actually need X", "Does Y cover it?", "Do you mean Z?", etc.
- Understand the problem: read the task and the code it touches, trace the real flow end to end.
- Follow the simple design priorities: tests pass, no duplicated knowledge, intent expressed, fewest elements.
- Avoid the loopholes: follow the instructions in [Anti-Loopholes](#anti-loopholes).
- Write code that is safe and maintainable: see the later sections, [Writing](#writing), [Fixing](#fixing), [Testing](#testing).
- Verify every API, function, option, and config key against this codebase and installed versions; never trust memory.
- Report honestly: what actually ran, never claim unverified success or present a stub or placeholder as finished; anything that could not run is named with its remaining risk.
<!-- Project-specific / Guideline -->
- Viction integration is add-ons, Ethereum logic must be intact after implementation. Consult the user if breaking changes is unavoidable.
- Prefer addition over overwrite existing existing Ethereum logic to reduce diffs that make merging upstream changes harder.

### Exploring

When exploring for solution for the task, try to reuse existing code following this priority order and stop at the first one that is satisfied; this step is needed to make the project easy to maintain and understandable in the long run:

- Does it already exist in this codebase? Reuse the helper, util, or pattern that's already here; don't rewrite it.
- Does the standard library already do this? Use it.
- Does a native platform feature cover it? Use it.
- Does an already-installed dependency solve it? Use it.
- If none of the above applies, writing new code is fine.
<!-- Project-specific / Reading -->

### Writing

When writing code, follow these rules:

- Explore carefully before writing, make use of what is available in the codebase first; see [Exploring](#exploring).
- Write the minimum that satisfies the task; never cut safety: validation, error handling, security, accessibility; see [Security](#security).
- Make targeted edits, never whole-file regeneration; every changed line should trace to the request.
- Keep one job per unit at every scale (function, type, package); if describing its job needs "and", split it. Parsing, domain rules, persistence, external calls, presentation, and wiring stay in separate homes.
- Use the narrowest access modifier; keep the public API surface small.
- Report unrelated findings; never do drive-by renames, reformats, or dependency bumps.
- Prefer small, focused functions and files; avoid large "god" packages or files.
- Prefer surrounding code's local, idiomatic style over general rules unless it is unsafe or broken.
- Prefer early exits for invalid or terminal cases over nested conditionals.
- Prefer allowlists over denylists.
- Remove what your change orphaned (unused code, imports, tests, files).
- When assigning values to object fields, follow the order of field declarations if possible.
- When renaming types, methods, functions, remember to check relevant tests.
- When moving files, use the source control move command to retain history.
- Avoid magic numbers or strings, use a named constant or inline comment to explain its meaning or *why* comment.
- Avoid sibling variants (`_v2`, `_new`, `_final`, `_copy`), replace the original.
- Avoid unrelated code in `utils`/`helpers`/`common`; name the domain concept instead.
- Avoid unnecessary abstraction, don't introduce an interface until there are two or more real implementations.
<!-- Project-specific / How to Editing -->

### Security

- Use parameterized queries and safe APIs.
- Keep authorization checks beside the operation they protect, or centralized in one enforced policy layer.
- Perform input validation at trust boundaries. Boundaries are external APIs, databases, file systems, clocks, queues, UI events, network calls, subprocesses, generated code.
- Prefer well-maintained standard libraries for crypto, parsing, auth, and serialization.
- Never log secrets, tokens, and sensitive data in error messages and logs.
- Never modify arguments of functions/methods as side effects; return values instead.
- Never hardcode secrets or commit them to the repository; read them from the environment or a secret store.
<!-- Project-specific / Security -->

### Fixing

When fixing issues, instructions of [Writing](#writing) apply, plus:

- Fix the root cause, not just the symptom.
- Write a test that reproduces the bug first, watch it fail, then make it pass.
- Grep every caller of the function you touch to make sure we don't introduce a new bug or leave a bug half-fixed.
- For refactor, run the relevant tests before and after; behavior must not change.
- For deduplication, only merge true duplication (copies that must always change together); merging accidental lookalikes is harder to undo than leaving them apart.
<!-- Project-specific / Fixing -->

### Testing

When writing tests, follow these rules:

- Write one concept per test; split a test whose name needs "and"; prefer behavior-focused names.
- Write at least three deterministic expectations, and make sure at least one fails on the untouched fixture.
- Keep the three parts of a test visibly distinct: arrange, act, assert.
- Keep tests F.I.R.S.T.: fast, independent, repeatable, self-validating, timely.
- Keep the deterministic results: no sleeps or timing guesses, no real network or filesystem call.
- Assert on outcomes, not implementation.
- Never weaken, skip, or delete failing tests; restore the test, fix the code, or report the conflict and stop.
<!-- Project-specific / Testing -->
- Before running tests that use the fixtures in `tests/testdata`, update the git submodule to the revision pinned in the commit tree (not in a file): `git submodule update --init --checkout tests/testdata`.

### Agent Smells

Check your diff against these failure patterns of AI-generated code; each must be fixed before completion:

- Hallucinated API: a function, method, option, or config key not in this codebase or the installed version => verify at the installed version, replace or remove.
- Unverified dependency: a new package or import without checking its registry name, version, or whether an installed dependency does the job => confirm the exact name/version (an invented name may be attacker-registered); prefer what is installed.
- Context loss: a stale or partial read that contradicts existing code, restores deleted code, or drops error handling in a rewrite => re-read target files, callers, tests.
- Scope creep: changed lines that trace to no request such as drive-by renames, reformatting, dependency bumps => revert them and report instead.
- Duplicate implementation: a helper or sibling variant paralleling one that exists => search before writing, extend the original, merge only true duplication, never code owned by different actors.
- Wrong-file gravity: logic added to whatever file was open, a god file growing, a new file at the repository root or current directory => place by role per the project layout, see [project-structure.md](project-structure.md).
- Phantom success: "should work now" with nothing run; a stub, pass, placeholder, or demo value presented as done => run the check and quote the result, name what did not run or state plainly what remains.
- Test weakening: an assertion loosened; a test skipped, deleted, or rewritten to match the bug, a snapshot re-accepted unread => restore the test and fix the code, or report the conflict and stop.
- Silent architecture drift: an outward import; an ORM, framework, or HTTP type in a business rule; a skipped layer; a new cycle; structure named after the stack.

### Anti-Loopholes

Stop and reassess when you catch yourself thinking:

| Rationalization | Reality |
| --- | --- |
| "I will clean this up while I am here." | Unrelated work unless the task needs it; report it instead. |
| "A framework will make this cleaner." | A dependency is a cost, and a one-sided commitment; prove the need. |
| "This abstraction will help later." | Later requirements can pay for later abstraction. |
| "The code is bad, so a rewrite is cleaner." | Rewrites need scope, tests, migration risk control — the team that made the mess usually rebuilds it. |
| "There are no tests, so verification is impossible." | Use the best available check and report remaining risk. |
| "I will put it here for now." | "For now" placements become permanent. Place it correctly once. |
| "The user asked for cleanup, so everything is in scope." | Campaign mode has a protocol: baseline, batches, ledger, verification. |
| "Clean code means following this skill over local style." | Local, idiomatic style wins unless unsafe or broken. |
| "It is only one import; the layering still basically holds." | One inward-facing name is the violation. Layering is a rule, not a tendency. |
| "Splitting it into services will decouple it." | A process boundary is not a boundary; shared data still couples through it. |
| "These two blocks are identical, so I will extract a helper." | Only if they must always change together. Check who owns each one. |
| "We will clean it up after the deadline." | The pressure that created the shortcut never abates. |

## Checklist

Before considering any coding task complete, please verify the following:

- [ ] Whole project is built successfully (see [commands.md](commands.md)).
- [ ] Code files are formatted, passed static analysis (see [commands.md](commands.md)).
- [ ] Changed code files follow project convention (see [coding-conventions.md](coding-conventions.md)).
- [ ] Imports are tidy (see [commands.md](commands.md)).
- [ ] Changes are scoped to the task, unrelated fixes are reported, not bundled in.
- [ ] Changes meet [Security](#security) requirements.
- [ ] Review the changes for common mistakes: missing validation, missing error handling, edge-cases, security vulnerability...
- [ ] Review the changes for agent smells (see [Agent Smells](#agent-smells)).
- [ ] New/changed behavior has test coverage.
- [ ] Relevant tests are passing, except the ones documented as known failures.
- [ ] No secrets, tokens, or sensitive payloads in logs or error messages.
- [ ] No unused code, debug prints, or commented-out blocks left behind.
- [ ] Report honestly: what actually ran, never claim unverified success or present a stub or placeholder as finished; anything that could not run is named with its remaining risk.
<!-- Project-specific / Checklist -->

## Project-specific

<!-- Project-specific -->
