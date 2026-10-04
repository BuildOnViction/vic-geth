# Coding Conventions

## General

Examples below use camelCase; adapt casing to the language's style.

### Formats

- Keep file content reads top-down: high-level intent first, details descending (newspaper metaphor).
- Keep related things close: declarations near first use, callers near callees.
- Start every code file, including test files, with the license header defined in [license-header.md](license-header.md).
<!-- Project-specific / Formats -->

### Names

- Use one word per concept across the codebase; never one word for two concepts.
- Use intention-revealing names: the name says why it exists and what it does, with no mental mapping from a terse abbreviation to a concept.
- Include units in names where values carry them (e.g. `timeoutMs`, `maxBytes`).
- Function names: start with a verb, and disclose side effects (`save`, `delete`, `publish`, or `mutate`).
- Acronyms: stay consistently cased (e.g. `ID`, `URL`, `HTTP`, not `Id`, `Url`, `Http`).
- Booleans: name as predicates (`isReady`, `hasItems`, `canRetry`).
- Avoid generic names: (`data`, `manager`, `helper`, `util`, `info`, `result`) when a domain name exists.
- Avoid noise: names that differ only by noise words or loose abbreviations.
<!-- Project-specific / Names -->

### Comments

- Document every exported identifier (constants, vars, types, methods, functions) with a doc comment.
- Write comments explaining *why*, not *what*; the code already shows what it does.
- Cut articles, filler, and hedging; keep IDs, numbers, links, and warnings whole.
- Keep comments short and concise, should be one to three lines. Longer explanations belong in a doc file the comment points to, or an extracted function.
- Keep comments up to date with the code, a stale comment is worse than no comment.
- Keep comments about: non-obvious constraints, legal requirements, algorithmic tradeoffs, external system quirks, warnings of consequences, and TODOs with context and ownership.
- Delete comment noise: redundant restatements, change journals, attributions, banner or position markers, closing-brace labels, and commented-out code.
- Never delete or reword comments a tool reads (shebangs, encoding lines, license and SPDX headers, build and compiler directives, suppressions, doctests, generated-file headers). An unexplained suppression is a finding, not a comment to delete.
<!-- Project-specific / Comments -->
- Function comment is an exception, it start with a verb and summary the intent of the function.

### Errors

- Handle errors explicitly where a meaningful decision can be made.
- Wrap errors with original cause and context when propagating.
- Error strings should be lowercase and not end with punctuation (they're often composed into larger messages).
- Make retry, fallback, timeout, and cancellation behavior explicit.
- Never log secrets, tokens, and sensitive data in error messages and logs.
- Never swallow errors or catch broadly without handling, wrapping, or propagating with context.
<!-- Project-specific / Errors -->

## Go

### Formats

- Run `gofmt -l .` to format the code; it must report no files.
- Run `goimports` to keep imports sorted and grouped: stdlib, then third-party, then local packages, separated by a blank line.
<!-- Project-specific / Go-Formats -->

### Names
- Package names: short, lowercase, single word, no underscores or mixedCaps.
- Interface names: prefer the `-er` suffix for single-method interfaces (e.g. `Reader`, `Closer`).
- Avoid stuttering: don't repeat the package name in exported identifiers (e.g. `log.Logger`, not `log.LogLogger`).
<!-- Project-specific / Go-Names -->

### Comments
- Refer [Go doc conventions](https://go.dev/doc/comment) for specific convention.
<!-- Project-specific / Go-Comments -->

## Project-specific

<!-- Project-specific -->
