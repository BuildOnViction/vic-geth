# AGENTS.md - Vic-Geth

This file is the entry point for AI coding assistants working on this repository. It is an index to detailed topic guides in the `.agents/` and `.attunic/` directories. Read the `.attunic/` files in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order) before work; from `.agents/`, read the file(s) relevant to the task at hand, you don't need to load all of them.

Project-specific rules supplement the template rules; put them below the `<!-- Project-specific / X -->` marker of the corresponding section, or below the final `<!-- Project-specific -->` marker for general additions.

## Topics

| File | When to read |
|------|--------------|
| [.agents/attunic-protocol.md](.agents/attunic-protocol.md) | Before reading or writing `.attunic/` files: project memory files, read order, write rules |
| [.agents/coding-conventions.md](.agents/coding-conventions.md) | Before writing or reviewing code: formatting, naming, packages, types, functions, comments, errors, concurrency, plus per-language and per-framework conventions |
| [.agents/development-guideline.md](.agents/development-guideline.md) | Before any coding task: exploring, writing, security, fixing, refactoring and campaigns, testing, common mistakes, checklist, plus per-language and per-framework sections |
| [.agents/license-header.md](.agents/license-header.md) | Before creating a code file: license header to put at its top |
| [.agents/memory-template.md](.agents/memory-template.md) | When an audit or cleanup campaign starts: skeleton for `wip.md` |
| [.agents/project-structure.md](.agents/project-structure.md) | When placing a file that [context.md's Layout](.attunic/context.md#layout) doesn't cover: standard project structures of languages and frameworks |
| [.agents/pull-request-guideline.md](.agents/pull-request-guideline.md) | Before opening or updating a pull request: guideline and checklist |
| [.attunic/context.md](.attunic/context.md) | Every session, first: technical specifications, file layout, layer dependencies, tool usage, confirmed facts |
| [.attunic/memory.md](.attunic/memory.md) | Before proposing a design change: append-only log of confirmed exceptions, deferrals, and open questions |
<!-- Project-specific / Topics -->

## Core Rules

Important rules when working with the project; these must be strictly followed:

- Follow instructions: anything explicitly requested, plus the instructions in [development-guideline.md](.agents/development-guideline.md).
- Resolve conflicts by priority: user requests, project-specific rules, local idiomatic style (unless unsafe or broken), framework sections, language sections, general rules.
- Read rule strength from its verb: a rule stated as a plain command (Keep, Use, Write, ...) is mandatory, and so are Always and Must; Never, Avoid, and Do not (don't) are mandatory prohibitions; Prefer is a default, deviate only with a stated reason.
- Load project memory first: read the `.attunic/` files in [attunic-protocol.md's Read Order](.agents/attunic-protocol.md#read-order) before work; [context.md's Confirmed Facts](.attunic/context.md#confirmed-facts) are binding, and a change to [context.md](.attunic/context.md) or [memory.md](.attunic/memory.md) is written only after the user explicitly confirms that change (see [attunic-protocol.md's Write Rules](.agents/attunic-protocol.md#write-rules)).
- Plan first: plan by default, unless the user explicitly says to skip it; when the change spans more than one file, share the plan and wait for the user's approval before the first edit.
- Ask when in doubt: if you are unsure about an instruction, need help, or have a better solution, ask: "Do you actually need X?", "Does Y cover it?", "Do you mean Z?", etc.
- Understand the problem: read the task and the code it touches, trace the real flow end to end.
- Follow the simple design priorities: tests pass, no duplicated knowledge, intent expressed, fewest elements.
- Avoid common loopholes: stop and reassess when you catch yourself thinking any of the rationalizations in [Anti-Loopholes](#anti-loopholes).
- Write clean code: safe and maintainable, per [coding-conventions.md](.agents/coding-conventions.md).
- Never trust recall: verify every API, function, option, and config key against this codebase and installed dependency versions.
- Respect template ownership: never edit content above a `Project-specific` marker, move or duplicate a marker, or edit `.copier-answers.yml`; `copier update` owns them.
- Confirm irreversible actions: ask before deleting data or files outside the change, rewriting history, force-pushing, publishing, or migrating shared environments.
- Commit and push only on request: never create a commit or push unless the user asks for it.
- Report fully: say what ran and what didn't, what was skipped or assumed, and what was found along the way; never present unverified success or a stub or placeholder as finished; name anything that could not run with its remaining risk.
<!-- Project-specific / Core Rules -->
- Viction integration is add-ons, Ethereum logic must be intact after implementation. Consult the user if breaking changes is unavoidable.
- Prefer addition over overwrite existing existing Ethereum logic to reduce diffs that make merging upstream changes harder.

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
| "The user asked for cleanup, so everything is in scope." | Campaign mode has a protocol: baseline, batches, wip, verification; see [Refactoring](.agents/development-guideline.md#refactoring). |
| "Clean code means following this document over local style." | Local, idiomatic style wins unless unsafe or broken. |
| "It is only one import; the layering still basically holds." | One outward import is the violation. Layering is a rule, not a tendency. |
| "Splitting it into services will decouple it." | A process boundary is not a boundary; shared data still couples through it. |
| "These two blocks are identical, so I will extract a helper." | Only if they must always change together. Check who owns each one. |
| "We will clean it up after the deadline." | The pressure that created the shortcut never abates. |
<!-- Project-specific / Anti-Loopholes -->

## Project-specific

<!-- Project-specific -->
