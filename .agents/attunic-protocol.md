# Attunic Protocol

This file defines `.attunic/`, the project memory layer: markdown-only, no scripts, read and edited like any other documentation. It lets a memoryless session resume instead of guessing, and keeps design intent written down where the next session can find it.

## Files

| File | Holds | Created by | Read when |
|------|-------|------------|-----------|
| [context.md](../.attunic/context.md) | Abstract, Technical Specifications, Layout, Layer Dependencies with their Deliberate Exceptions, Supporting Tools, Confirmed Facts of the project. | Generated with the project; `Confirmed Facts` grows during work | Every session, first |
| [memory.md](../.attunic/memory.md) | Decisions made and why (append-only); open questions awaiting answers | Generated with the project; grows during work | Before proposing a design change |
| [wip.md](../.attunic/wip.md) | Audit findings, technical debt | Created on demand when an audit starts; gitignored, not committed; may not exist yet | Before starting or resuming one |
<!-- Project-specific / Files -->

## Read Order

At the start of a session, before the first edit:

1. [context.md](../.attunic/context.md).
2. [memory.md](../.attunic/memory.md).
3. [wip.md](../.attunic/wip.md), if it exists.
<!-- Project-specific / Read Order -->

## Write Rules

- Write only what the next session cannot re-derive from the code: record the reasoning, not what the code plainly shows.
- Append only to [memory.md](../.attunic/memory.md): supersede an entry by adding a new one that references it.
- Record lasting choices:
  - A decision between real alternatives, a deliberate deferral, or a constraint discovered the hard way goes to [memory.md](../.attunic/memory.md).
  - A fact a person confirmed goes to [context.md](../.attunic/context.md)'s `Confirmed Facts` section.
- Keep [wip.md](../.attunic/wip.md) current: an accurate work-in-progress list with an interrupted campaign is a good outcome; a finished campaign with a stale one is not.
- Never invent history: an unknown "why" is recorded under [memory.md](../.attunic/memory.md)'s `Open Questions`, not as a plausible reason; a confident wrong entry is worse than none.
- Never create or edit an `.attunic/` file silently: making changes to any of these files changes how the project is run, so it needs the user's consent.
<!-- Project-specific / Write Rules -->

## Creating Files On Demand

To create `wip.md`, copy the skeleton in [memory-template.md](memory-template.md).
<!-- Project-specific / Creating Files On Demand -->

## Project-specific

<!-- Project-specific -->
