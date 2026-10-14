# AGENTS.md - Vic-Geth

This file is the entry point for AI coding assistants working on this repository. It is an index to detailed topic guides in the `.agents/` and `.attunic/` directories. Read the relevant file(s) for the task at hand; you don't need to load all of them.

Project-specific rules supplement the template rules; put them below the `<!-- Project-specific / X -->` marker of the corresponding section.

## Topics

| File | When to read |
|------|---------------|
| [.agents/attunic-protocol.md](.agents/attunic-protocol.md) | Project memory layer in `.attunic/`: read before work, record decisions, resume campaigns |
| [.agents/coding-conventions.md](.agents/coding-conventions.md) | Naming, formatting, comment style, and other conventions |
| [.agents/development-guideline.md](.agents/development-guideline.md) | Guideline and checklist for working on this codebase (exploring, writing, fixing, testing) |
{%- if license != 'None' and license_header != 'None' %}
| [.agents/license-header.md](.agents/license-header.md) | License header to put at the top of every code file |
{%- endif %}
| [.agents/memory-template.md](.agents/memory-template.md) | Skeleton for `wip.md` when an audit or cleanup campaign starts |
| [.agents/project-structure.md](.agents/project-structure.md) | Standard project structures of languages and frameworks for reference |
| [.agents/pull-request-guideline.md](.agents/pull-request-guideline.md) | Guideline and checklist to run through before opening/updating a pull request |
| [.attunic/context.md](.attunic/context.md) | Project memory attuned specifically for this project: technical specifications, file layout, layer dependencies, tool usage; read before work |
| [.attunic/memory.md](.attunic/memory.md) | Append-only log of settled decisions and open questions; read before proposing a design change |
<!-- Project-specific / Topics -->

## Core Rules

Important rules when working with the project; these must be strictly followed:

- Follow instructions: anything explicitly requested, plus the instructions in [development-guideline.md](.agents/development-guideline.md); priority order: user requests, project-specific, template.
- Load project memory first: read `.attunic/context.md` and the other existing `.attunic/` files before work (see [attunic-protocol.md](.agents/attunic-protocol.md)); recorded decisions are settled, and lasting choices are appended to `.attunic/memory.md`.
- Plan first: plan by default, unless the user explicitly says to skip it.
- Ask when in doubt: if you are unsure about an instruction, need help, or have a better solution, feel free to ask questions: "Do you actually need X?", "Does Y cover it?", "Do you mean Z?", etc.
- Understand the problem: read the task and the code it touches, trace the real flow end to end.
- Follow the simple design priorities: tests pass, no duplicated knowledge, intent expressed, fewest elements.
- Avoid common loopholes: stop and reassess when you catch yourself thinking any of the rationalizations in [Anti-Loopholes](#anti-loopholes).
- Write clean code: safe and maintainable, per [development-guideline.md](.agents/development-guideline.md).
- Never trust memory: verify every API, function, option, and config key against this codebase and installed dependency versions.
- Report honestly: say what actually ran; never claim unverified success or present a stub or placeholder as finished; anything that could not run is named with its remaining risk.
<!-- Project-specific / Core Rules -->
- Viction integration is add-ons, Ethereum logic must be intact after implementation. Consult the user if breaking changes is unavoidable.
- Prefer addition over overwrite existing existing Ethereum logic to reduce diffs that make merging upstream changes harder.

## Anti-Loopholes

Stop and reassess when you catch yourself thinking:

| Rationalization | Reality |
| --- | --- |
| "I will clean this up while I am here." | Unrelated work unless the task needs it; report it instead. |
| "A framework will make this cleaner." | A dependency is a cost, and a one-sided commitment; prove the need. |
| "This abstraction will help later." | Later requirements can pay for later abstraction. |
| "The code is bad, so a rewrite is cleaner." | Rewrites need scope, tests, migration risk control. |
| "There are no tests, so verification is impossible." | Use the best available check and report remaining risk. |
| "I will put it here for now." | "For now" placements become permanent. Place it correctly once. |
| "The user asked for cleanup, so everything is in scope." | Campaign mode has a protocol: baseline, batches, wip, verification. |
| "Clean code means following this document over local style." | Local, idiomatic style wins unless unsafe or broken. |
| "It is only one import; the layering still basically holds." | One inward-facing name is the violation. Layering is a rule, not a tendency. |
| "Splitting it into services will decouple it." | A process boundary is not a boundary; shared data still couples through it. |
| "These two blocks are identical, so I will extract a helper." | Only if they must always change together. Check who owns each one. |
| "We will clean it up after the deadline." | The pressure that created the shortcut never abates. |
<!-- Project-specific / Anti-Loopholes -->

## Project-specific

<!-- Project-specific -->
