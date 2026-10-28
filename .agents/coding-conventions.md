# Coding Conventions

## General

The principles below are for all languages. They use camelCase for demonstration purposes; adapt casing to the language's style.

Language and framework sections refine them; on conflict, follow the priority order in [AGENTS.md's Core Rules](../AGENTS.md#core-rules).

When a language's established idiom contradicts a principle below, follow the idiom even when its section doesn't name it (e.g. PowerShell's `-Force` switch parameter, Python's keyword-only `reverse=True` flag).

### Formats

- Let the project's formatter own the format: run the formatter declared in [context.md's Supporting Tools](../.attunic/context.md#supporting-tools), if any; its check mode must report no file to change.
- Follow `.editorconfig` for indentation, line endings, charset, and final newline; it wins over editor and community defaults (e.g. tab indentation where a language has no official rule).
- Use ASCII punctuation and straight quotes by default in code, comments, and docs; use non-ASCII characters only where the content requires them (names, quoted text, symbols with no ASCII form, box-drawing in layout trees).
- Order file content top-down: high-level intent first, details descending (newspaper metaphor).
- Keep related things close: declarations near first use, callers near callees.
- Start every new code file, including test files, with the license header defined in [license-header.md](license-header.md).
<!-- Project-specific / Formats -->

### Names

- Use one word per concept across the codebase; never one word for two concepts.
- Use intention-revealing names: the name says why it exists and what it does, with no mental mapping from a terse abbreviation to a concept.
- Include units in names where the type does not carry them (e.g. `timeoutMs`, `maxBytes`); a value typed as a duration or size needs none.
- Start the name of a function that performs an action with a verb that discloses its side effects (`save`, `delete`, `publish`, `mutate`); name a side-effect-free accessor for what it returns (`owner`, `totalPrice`).
- Case acronyms consistently, following the language's convention; never mix forms in one codebase (`userId` beside `userID`).
- Name booleans as predicates (`isReady`, `hasItems`, `canRetry`), never as negations (`isNotReady`).
- Avoid magic numbers or strings; use a named constant for the meaning, or attach an inline *why* comment.
- Avoid noise: names that differ only by noise words or loose abbreviations.
- Never write a vague name (`data`, `info`, `obj`, `temp`, `result`) when a domain name exists; a loop index, a lambda or callback parameter, a caught error, or an established short form may stay short; nothing else may.
- Never write a bare-verb function with no object (`handle`, `process`); a method's receiver counts as its object (`file.close()`).
- Never number or version a name instead of replacing the original (`user2`, `dataV2`, `_new`, `_final`, `_copy`).
- Never suffix a type with a noise word (`Manager`, `Helper`, `Util`) or name it after a verb (`ProcessOrder`).
- Never encode a type or scope into a name (`strName`, `arrItems`, `iCount`).
<!-- Project-specific / Names -->

### Packages

- Default to the narrowest visibility; export only what another module must use.
- Name a module for the domain concept it owns; never add to a catch-all `utils`, `helpers`, or `common`.
<!-- Project-specific / Packages -->

### Types

- Avoid unnecessary abstraction: introduce an interface, trait, protocol, or abstract base only once two implementations exist; a test double counts as one.
- Assign object fields in their declaration order where possible.
<!-- Project-specific / Types -->

### Functions

- Keep functions small, at one level of abstraction; extract a named function instead of nesting another branch or commenting a block.
- Prefer early exits for invalid or terminal cases over nested conditionals.
- Keep positional parameters to three or fewer; past that, group related values into one parameter object.
- Never switch behavior on a boolean parameter; write two functions or take a small enum.
- Never mutate an input the caller does not expect to change; return a new value instead, and when mutation is the contract (an in-place sort, a buffer fill), make it visible in the name or signature.
<!-- Project-specific / Functions -->

### Comments

- Document every exported identifier (constants, vars, types, methods, functions) with a doc comment stating its contract: what it is or does, and what a caller must know.
- Write inline comments explaining *why*, not *what*; the code already shows what it does.
- Cut filler and hedging from comments, and articles from inline comments; keep IDs, numbers, links, and warnings whole.
- Keep inline comments short: one to three lines; longer explanations belong in a doc file the comment points to, or an extracted function.
- Keep comments up to date with the code: a stale comment is worse than no comment.
- Keep comments about: non-obvious constraints, legal requirements, algorithmic tradeoffs, external system quirks, warnings of consequences, and TODOs with context and ownership.
- Never write comment noise: redundant restatements, change journals, attributions, banner or position markers, closing-brace labels, and commented-out code; in existing code, report noise instead of deleting it, unless the task is a cleanup.
- Never delete or reword comments a tool reads (shebangs, encoding lines, license and SPDX headers, build and compiler directives, suppressions, doctests, generated-file headers). An unexplained suppression is a finding, not a comment to delete.
<!-- Project-specific / Comments -->

### Errors

- Handle each error once, where a meaningful decision can be made; elsewhere propagate it, and never log and rethrow it at every frame.
- Wrap errors with original cause and context when propagating.
- Model an expected outcome (not found, declined, empty) as a value, an optional, or a result type, never a sentinel value (`-1`, `""`); reserve the language's error mechanism for genuine failures.
- Reserve crashes (panic, abort, process exit) for programmer errors the caller cannot act on; never crash library code on an expected failure.
- Write user-facing error messages as actionable sentences; never expose internals (stack traces, queries, file paths) in them.
- Make retry, fallback, timeout, and cancellation behavior explicit.
- Never swallow errors or catch broadly without handling, wrapping, or propagating with context.
<!-- Project-specific / Errors -->

### Concurrency

- Give every concurrent task an owner that can wait for it or cancel it; never start one that nothing can stop.
- Never block an event loop or async runtime with blocking I/O or long computation; hand it to an async API, a worker, or a blocking-task pool.
- Keep shared mutable state to a minimum; hold a lock as briefly as possible, and never across a suspension point.
- Treat every suspension point as a place where shared state can change; re-check invariants after it.
<!-- Project-specific / Concurrency -->

## Go

### Formats

- Keep imports sorted and grouped: standard library, then third-party, then local packages, separated by a blank line.
<!-- Project-specific / Go-Formats -->

### Names

- Use MixedCaps (exported) or mixedCaps (unexported), since case marks visibility; never underscores.
- Name packages short, lowercase, and single-word, with no underscores or mixedCaps; never `util`, `common`, or `base`.
- Name a getter after the value it returns, without a `Get` prefix (e.g. `Owner`, not `GetOwner`).
- Keep receiver names short (one or two letters) and identical across every method of the type.
- Keep an initialism in one case throughout (`ID`, `URL`, `HTTP`), e.g. `userID`, `ServeHTTP`; never `userId` or `ServeHttp`.
- Prefer the `-er` suffix for a single-method interface (e.g. `Reader`, `Closer`); never an `I` prefix.
- Never repeat the package name in an exported identifier (e.g. `log.Logger`, not `log.LogLogger`).
<!-- Project-specific / Go-Names -->

### Packages

- Keep one package per directory; never split a single package's files across unrelated directories.
- Keep `init()` to registration only; never reach out to a database, file, or network from one.
<!-- Project-specific / Go-Packages -->

### Types

- Accept an interface as a parameter, return a concrete struct; declare the interface at the consumer that needs it, never beside the implementation.
- Guard a type whose zero value is unsafe to use behind a constructor (`NewX`); document the zero value when it is safe.
<!-- Project-specific / Go-Types -->

### Functions

- Keep the success path unindented and last, returning early on each error or unmet precondition.
- Use named returns only to document a short function's intent, never to smuggle a late, hidden mutation.
<!-- Project-specific / Go-Functions -->

### Comments

- Refer to [Go doc conventions](https://go.dev/doc/comment) for specific convention.
- State a package's purpose once, in its doc comment; do not restate it per file.
<!-- Project-specific / Go-Comments -->
- Function comment is an exception, it start with a verb and summary the intent of the function.

### Errors

- Wrap with context and keep the cause: `fmt.Errorf("loading order %s: %w", id, err)`.
- Compare with `errors.Is` and extract with `errors.As`; never compare an error's `.Error()` string.
- Return `(T, error)` rather than a sentinel zero value standing in for failure.
- Report an expected miss with the comma-ok form `(T, bool)` when absence is the only other outcome, and with an exported sentinel error (`ErrNotFound`) matched by `errors.Is` when the call can also fail for other reasons.
- Write error strings lowercase, without trailing punctuation; they are composed into larger messages.
- Check every returned error where it is returned; assigning one to `_` outside a test is a defect, not a shortcut.
<!-- Project-specific / Go-Errors -->

### Concurrency

- Give every goroutine an owner for its shutdown; never start one with `go func() { ... }()` and no way to wait for or cancel it.
- Take `context.Context` as the first parameter and check `ctx.Done()`/`ctx.Err()` on any path that can block; never store it on a struct field.
- Run goroutines that share a lifetime and must fail together under one group that waits for all and returns the first error, rather than a `sync.WaitGroup` plus a hand-rolled error slice.
- The sender closes a channel, never the receiver; closing one with a live sender still writing to it panics.
- Keep package-level state immutable; mutable package state read or written from more than one goroutine is a defect.
- Copy a loop variable (`v := v`) only where the `go` directive in `go.mod` still selects one variable shared by every iteration; otherwise each iteration already owns its own.
<!-- Project-specific / Go-Concurrency -->

## Project-specific

<!-- Project-specific -->
