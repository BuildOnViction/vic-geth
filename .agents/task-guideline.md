# Task Guideline

## Guideline

Important rules when working with the project; these must be strictly followed:

- Follow instructions: anything that is explicitly requested.
- Planning first: plan by default, unless the user explicitly says to skip it.
- Ask when in doubt: if you are unsure about an instruction, need help, or have a better solution, feel free to ask questions: "Do you actually need X", "Does Y cover it?", "Do you mean Z?", etc.
- Understand the problem: read the task and the code it touches, trace the real flow end to end.
- Write code that is maintainable: see the later sections `Sharing Code`, `Editing`, `Fixing`, `Testing`.
<!-- Project-specific / Guideline -->
- Viction integration is add-ons, Ethereum logic must be intact after implementation. Consult the user if breaking changes is unavoidable.
- Prefer addition over overwrite existing existing Ethereum logic to reduce diffs that make merging upstream changes harder.

### Sharing Code

When exploring the codebase for solution, try to reuse existing code following this priority order and stop at the first one that is satisfied; this step is needed to make the project easy to maintain and understandable in the long run:

- Does this need to be built at all? YAGNI.
- Does it already exist in this codebase? Reuse the helper, util, or pattern that's already here; don't rewrite it.
- Does the standard library already do this? Use it.
- Does a native platform feature cover it? Use it.
- Does an already-installed dependency solve it? Use it.
- If none of the above applies, writing new code is fine.
<!-- Project-specific / Sharing Code -->

### Editing

When adding/modifying code, try to follow these instructions:

- For each package, follow the convention of existing files first.
- If the package is new, or when adding new code, the preferred ordering in a file is: for each type, its declaration followed by its own methods (public before private), repeated per type in the file; then standalone functions (public before private) at the end.
- When assigning values to object fields, try to follow the order of field declarations if possible.
- Perform input validation at trust boundaries.
- Allowlists preferred over denylists.
- If moving files is needed, use the source control move command to retain history.
- When renaming types, functions, remember to check relevant tests.
<!-- Project-specific / Editing -->

### Fixing

If you are fixing issues, instructions of `Sharing Code`, `Editing` apply, plus:

- Fix the root cause, not the symptom.
- Grep every caller of the function you touch to make sure we don't introduce a new bug or leave a bug half-fixed.
<!-- Project-specific / Fixing -->

### Testing

When testing, follow these rules:

- Test functions must follow the order of the code they test.
- Execute tests per package to prevent timeout.
<!-- Project-specific / Testing -->
- Before running tests that use the fixtures in `tests/testdata`, update the git submodule to the revision pinned in the commit tree (not in a file): `git submodule update --init --checkout tests/testdata`.

## Checklist

Before considering any coding task complete, please verify the following:

- [ ] Whole project is built successfully (see [commands.md](commands.md)).
- [ ] Code files are formatted, passed static analysis (see [commands.md](commands.md)).
- [ ] Changed code files follow project convention (see [coding-conventions.md](coding-conventions.md)).
- [ ] Imports are tidy (see [commands.md](commands.md)).
- [ ] Review the changes for common mistakes: missing validation, missing error handling, edge-cases, security vulnerability...
- [ ] Relevant tests are passing, except the ones documented as known failures.
- [ ] New/changed behavior has test coverage.
- [ ] Changes are scoped to the task, no unrelated refactors bundled in.
- [ ] No unused code, debug prints, or commented-out blocks left behind.
<!-- Project-specific / Checklist -->

## Project-specific

<!-- Project-specific -->
