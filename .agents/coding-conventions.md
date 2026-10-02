# Coding / Naming / Comment Conventions

## General

- One package per directory; keep package names short and focused.
- Prefer small, focused functions and files; avoid large "god" packages or files.
- Start every code file, including test files, with the license header defined in [license-header.md](license-header.md).
- Avoid unnecessary abstraction, don't introduce an interface until there are two or more real implementations.
<!-- Project-specific / General -->

## Names

- Use `MixedCaps` or `mixedCaps`; never underscores in identifiers.
- Package names: short, lowercase, single word, no underscores or mixedCaps.
- Interface names: prefer the `-er` suffix for single-method interfaces (e.g. `Reader`, `Closer`).
- Acronyms stay consistently cased (`ID`, `URL`, `HTTP`), not `Id`, `Url`, `Http`.
- Avoid stuttering: don't repeat the package name in exported identifiers (e.g. `log.Logger`, not `log.LogLogger`).
<!-- Project-specific / Names -->

## Formats

- All Go source must be formatted with `gofmt` (tabs for indentation, not spaces). Run `gofmt -l .` before committing; it must report no files.
- Run `goimports` (or `gofmt` with import grouping) to keep imports sorted and grouped: stdlib, then third-party, then local packages, separated by a blank line.
<!-- Project-specific / Formats -->

## Comments

- Document every exported identifier (types, functions, constants, vars) with a doc comment that starts with the identifier's name, per [Go doc conventions](https://go.dev/doc/comment).
- Comments explain *why*, not *what* - the code already shows what it does.
- Keep comments short and concise, up to date with the code; a stale comment is worse than no comment.
<!-- Project-specific / Comments -->
- Function comment is an exception, it start with a verb and summary the intent of the function.

## Errors

- Handle errors explicitly (`if err != nil { ... }`); never silently discard them.
- Wrap errors with context using `fmt.Errorf("...: %w", err)` when propagating up.
- Error strings should be lowercase and not end with punctuation (they're often composed into larger messages).
<!-- Project-specific / Errors -->

## Project-specific

<!-- Project-specific -->
