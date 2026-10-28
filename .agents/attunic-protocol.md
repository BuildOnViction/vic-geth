# Attunic Protocol

This file defines `.attunic/`, the project memory: markdown-only, no scripts, read and edited like any other documentation. It lets a memoryless session resume instead of guessing, and keeps design intent written down where the next session can find it.

## Files

| File | Holds | Created by | Read when |
|------|-------|------------|-----------|
| [context.md](../.attunic/context.md) | Abstract, Technical Specifications, Layout, Layer Dependencies with their Deliberate Exceptions, Supporting Tools, Confirmed Facts of the project. | Generated with the project, then filled in by the user, by hand or with an agent's help on request | Every session, first |
| [memory.md](../.attunic/memory.md) | Confirmed exceptions and deferrals with their reasons (append-only), kept as a log, not as rules; open questions and their answers | Generated with the project; grows during work | Before proposing a design change |
| [wip.md](../.attunic/wip.md) | Audit findings, technical debt | Created on demand when an audit or cleanup campaign starts; gitignored, not committed; may not exist yet | Every session, after `context.md`, if it exists |
<!-- Project-specific / Files -->

## Read Order

Read these every session, first, before any other work:

1. [context.md](../.attunic/context.md).
2. [wip.md](../.attunic/wip.md), if it exists.

Treat a section with nothing below its marker as empty: it declares nothing and is not missing information; don't ask the user to fill it in, offer to draft it, or guess its content.
<!-- Project-specific / Read Order -->

## Write Rules

- Write only what the next session cannot re-derive from the code: record the reasoning, not what the code plainly shows.
- Append only to [memory.md](../.attunic/memory.md): supersede an entry by adding a new one that references it; the only in-place addition is an answer appended to its open question.
- Add an entry at the end of the section it belongs to, below its marker and after the entries already there (e.g. an exception or deferral below `<!-- Project-specific / Decisions -->`), never at the end of the file.
- When a task is finished, draft a [memory.md's Decisions](../.attunic/memory.md#decisions) entry for each exception (something that doesn't adhere to the rules but must be left as-is) or deferral (something deliberately left for later) the user confirmed during it, with its reason; then ask per the confirmation rule below.
- Leave [context.md](../.attunic/context.md), `Confirmed Facts` included, to the user: they fill it in by hand or explicitly ask you to help; never draft or propose its content unprompted.
- Treat [wip.md](../.attunic/wip.md) as transient memory of this workspace: once the user consents to proceed with a campaign, create it and keep it current without asking for each edit; an accurate work-in-progress list with an interrupted campaign is a good outcome, a finished campaign with a stale one is not.
- Close a campaign by reporting the open rows of [wip.md](../.attunic/wip.md)'s `Found But Not Fixed` and `Deferred` tables and proposing its deferrals for [memory.md's Decisions](../.attunic/memory.md#decisions), then delete `wip.md` once the user agrees; the next campaign starts from a fresh skeleton.
- Never invent history: an unknown "why" is recorded under [memory.md](../.attunic/memory.md)'s `Open Questions`, not as a plausible reason; a confident wrong entry is worse than none.
- Never write to [context.md](../.attunic/context.md) or [memory.md](../.attunic/memory.md) without the user's explicit confirmation of that change: these files change how the project is run, so show the drafted change and ask, in a question of its own, whether to record it; a blanket approval of a plan or of several items ("yes", "proceed", "do all") does not count, and neither does approving the decision or fact itself. Without that confirmation, name the unrecorded change in your report instead.
<!-- Project-specific / Write Rules -->

## Creating Files On Demand

To create `wip.md`, copy the skeleton in [memory-template.md](memory-template.md).
<!-- Project-specific / Creating Files On Demand -->

## Project-specific

<!-- Project-specific -->
