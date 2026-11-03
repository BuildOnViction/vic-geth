# AGENTS.md - Vic-Geth

This file is the entry point for AI coding assistants working on this repository. It holds the rules for every task in [Core Rules](#core-rules) and [Anti-Loopholes](#anti-loopholes), and indexes the detailed topic guides in the `.agents/` directory and the project memory in the `.attunic/` directory. Read each file in [Topics](#topics) when its `When to read` entry applies; you don't need to load all of them.

The files that come from this project's template are the files with `Project-specific` markers, plus `CLAUDE.md`, the license files at the repository root, and `.copier-answers.yml`, which have no marker and are template-owned in full.

A marker shows where its file can be customized. It ends the template-owned content of its section and is named after that section, uniquely within its file:

- In Markdown, it is an HTML comment on a line of its own: `<!-- Project-specific / <Section> -->`.
- In a table that ends a section, it is a row of its own, with the comment in its first cell.
- In a fenced layout tree that ends a section, it is an additional item of the tree, on a line of its own before its closing fence; ignore that item when reading the tree: it is not part of the tree's content.
- In `.editorconfig` and `.gitignore`, it is the `# Project-specific` comment line; these files have no sections, so it ends the template-owned content of the whole file.

A file with markers holds two kinds of content:

- Project-owned content: the content below a marker, where the project adds its customizations.
  - It runs down to the end of the file or the next Markdown heading that has as many or fewer `#` than the marker's section heading or has a marker of its own; in `.editorconfig` and `.gitignore`, it runs to the end of the file.
  - Give any heading you add there more `#` than the marker's section heading; a heading with as many or fewer `#` ends the project-owned content.
  - Add table rows below the table's marker row, in the same table.
  - Add tree items below the tree's marker line, in the same layout tree.
- Template-owned content: everything else in the file; `copier update` maintains it.

## Topics

| File | When to read |
|------|--------------|
| [.agents/attunic-protocol.md](.agents/attunic-protocol.md) | Every session, first, before reading or writing any `.attunic/` file: project memory files, read order, write rules |
| [.agents/coding-conventions.md](.agents/coding-conventions.md) | Before writing or reviewing code or docs: formatting, naming, packages, types, functions, comments, errors, concurrency, plus per-language and per-framework conventions |
| [.agents/development-guideline.md](.agents/development-guideline.md) | Before writing or reviewing code or docs: exploring, writing, security, fixing, refactoring and workstreams, testing, common mistakes, checklist, plus per-language and per-framework sections |
| [.agents/license-header.md](.agents/license-header.md) | Before creating a code file: license header to put at its top |
| [.agents/memory-template.md](.agents/memory-template.md) | When an audit or cleanup workstream starts: skeleton for `workstream.md` |
| [.agents/project-structure.md](.agents/project-structure.md) | When placing a file that [context.md's Layout](.attunic/context.md#layout) doesn't cover: standard project structures of languages and frameworks |
| [.agents/pull-request-guideline.md](.agents/pull-request-guideline.md) | Before opening or updating a pull request: guideline and checklist |
| [.attunic/context.md](.attunic/context.md) | Every session, in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order): abstract, technical specifications, layout, layer dependencies with their deliberate exceptions, supporting tools |
| [.attunic/memory.md](.attunic/memory.md) | Every session, in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order): confirmed facts, binding instructions unique to this project |
| [.attunic/question.md](.attunic/question.md) | Every session, in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order): open questions, each removed once answered |
| `.attunic/workstream.md` | Every session, if it exists, in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order): baseline, batches, found but not fixed, deferred of the current audit or cleanup workstream |
| <!-- Project-specific / Topics --> | |

## Core Rules

Important rules when working with the project; follow them strictly unless the conflict rules below let another source override one:

- Follow instructions: anything explicitly requested, plus the instructions in the files in [Topics](#topics), each when its `When to read` entry applies.
- Resolve conflicts by priority, highest first; [question.md](.attunic/question.md) and `workstream.md` hold open questions and work state, not rules, and take no part in it:
  1. User requests.
  2. This file.
  3. [memory.md](.attunic/memory.md).
  4. [context.md](.attunic/context.md).
  5. [attunic-protocol.md](.agents/attunic-protocol.md).
  6. Project-owned content of the other `.agents/` guides: their `## Project-specific` sections first, then framework, language, and general sections.
  7. Local idiomatic style in code, unless unsafe or broken.
  8. Template-owned content of the framework sections of the other `.agents/` guides.
  9. Template-owned content of the language sections of the other `.agents/` guides.
  10. Template-owned content of the general sections of the other `.agents/` guides.
- Resolve conflicts inside a file or section: inside a file, its `## Project-specific` section wins over its other sections; inside a section, its project-owned content wins over its template-owned content.
- Read rule strength from its verb: a rule stated as a plain command (Keep, Use, Write, ...) is mandatory, and so are Always and Must; Never and Do not (don't) are mandatory prohibitions; Prefer and Avoid are defaults, deviate only with a stated reason.
- Load project memory first: read the `.attunic/` files in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order) before work; [memory.md's Confirmed Facts](.attunic/memory.md#confirmed-facts) are binding, and a change to [context.md](.attunic/context.md), [memory.md](.attunic/memory.md), or [question.md](.attunic/question.md) is written only after the user explicitly confirms that change (see [attunic-protocol.md's Write Rules](.agents/attunic-protocol.md#write-rules)).
- Plan first: plan by default, unless the user explicitly says to skip it; when the change spans more than one file, share the plan and wait for the user's approval before the first edit.
- Ask when in doubt: if you are unsure about an instruction, need help, or have a better solution, ask: "Do you actually need X?", "Does Y cover it?", "Do you mean Z?", etc.
- Understand the problem: read the task and the code it touches, trace the real flow end to end.
- Follow the simple design priorities: tests pass, no duplicated knowledge, intent expressed, fewest elements.
- Guard against common loopholes: stop and reassess when you catch yourself thinking any of the rationalizations in [Anti-Loopholes](#anti-loopholes).
- Distrust recall: verify every API, function, option, and config key against this codebase and installed dependency versions.
- Respect template ownership: never edit template-owned content, add, move, or duplicate a marker; `copier update` owns them.
- Confirm irreversible actions: ask before deleting data or files outside the change, rewriting history, force-pushing, publishing, or migrating shared environments.
- Verify every change: every changed behavior is checked by something that runs; without a test harness, the smallest check that runs (a script or a command) stands in for each test a rule asks for.
- Commit and push only on request: never create a commit or push unless the user asks for it.
- Report fully: say what ran and what didn't, what was skipped or assumed, and what was found along the way; never present unverified success or a stub or placeholder as finished; name anything that could not run with its remaining risk.
<!-- Project-specific / Core Rules -->
- Respect Ethereum logic: Viction integration is add-ons, Ethereum logic must be intact after implementation, consult the user if breaking changes is unavoidable.

## Anti-Loopholes

Stop and reassess when you catch yourself thinking:

| Rationalization | Reality |
|-----------------|---------|
| "I will clean this up while I am here." | Unrelated work unless the task needs it; report it instead. |
| "A framework will make this cleaner." | A dependency is a cost, and a one-sided commitment; prove the need. |
| "This abstraction will help later." | Later requirements can pay for later abstraction. |
| "The code is bad, so a rewrite is cleaner." | Rewrites need scope, tests, migration risk control. |
| "There are no tests, so verification is impossible." | Use the best available check and report remaining risk. |
| "I will put it here for now." | "For now" placements become permanent. Place it correctly once. |
| "The user asked for cleanup, so everything is in scope." | Workstream mode has a protocol: baseline, batches, `workstream.md`, verification; see [Refactoring](.agents/development-guideline.md#refactoring). |
| "Clean code means following these guides over local style." | Local idiomatic style outranks the template-owned content of the other `.agents/` guides; see the conflict order in [Core Rules](#core-rules). |
| "It is only one import; the architecture layering still basically holds." | One outward import is the violation. Architecture layering is a rule, not a tendency. |
| "Splitting it into services will decouple it." | A process boundary is not a decoupling boundary; shared data still couples through it. |
| "These two blocks are identical, so I will extract a helper." | Only if they must always change together. Check who owns each one. |
| "We will clean it up after the deadline." | The pressure that created the shortcut never abates. |
| <!-- Project-specific / Anti-Loopholes --> | |

## Project-specific

<!-- Project-specific / Project-specific -->
