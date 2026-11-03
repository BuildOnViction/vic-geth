# Attunic Protocol

This file defines `.attunic/`, the project memory: markdown-only, no scripts, read and edited like any other documentation. It holds what an agent working on this project can't derive on its own, so no session has to guess, ask, or re-confirm it. It describes the project as it is now, not how it got there: it is kept current, and a stale entry is fixed or removed, never kept for the record.

## Files

| File | Holds | Created by |
|------|-------|------------|
| [context.md](../.attunic/context.md) | Abstract, Technical Specifications, Layout, Layer Dependencies with their Deliberate Exceptions, Supporting Tools of the project. | Generated with the project, then filled in by the user, by hand or with an agent's help on request |
| [memory.md](../.attunic/memory.md) | Confirmed Facts: binding instructions unique to this project, kept current by editing them in place | Generated with the project, then filled in by the user, by hand or with an agent's help on request |
| [question.md](../.attunic/question.md) | Open questions, each removed once answered | Generated with the project; changes during work |
| `workstream.md` | Baseline, Batches, Found But Not Fixed, Deferred of the current audit or cleanup workstream | Created on demand when an audit or cleanup workstream starts; gitignored, not committed; may not exist yet |
| <!-- Project-specific / Files --> | | |

## Read Order

Read these every session, after this file, before any other work:

1. [context.md](../.attunic/context.md).
2. [memory.md](../.attunic/memory.md).
3. [question.md](../.attunic/question.md).
4. `workstream.md`, if it exists.

Treat a section with nothing below its marker as empty: it declares nothing and is not missing information; don't ask the user to fill it in, offer to draft it, or guess its content.
<!-- Project-specific / Read Order -->

## Write Rules

- Write only what an agent can't derive from the project or its training knowledge, or only with a lot of time and effort; never restate what the code, its docs, or common practice already show, or record how a decision was reached.
- Add a question to [question.md](../.attunic/question.md) only if it isn't already there, and remove it once it is answered; an answer a later agent needs is reported like any other project knowledge (see below).
- Add an entry at the end of the section it belongs to, below its marker and after the entries already there (e.g. a question below `<!-- Project-specific / Open Questions -->`), never at the end of the file.
- When a task is finished, report each piece of project knowledge the user gave during it that a later agent couldn't derive and would otherwise have to guess or ask about, an exception (something that doesn't adhere to the rules but must be left as-is) included, with its reason, for the user to record in the project memory.
- When a task is finished, report each [memory.md](../.attunic/memory.md) fact or [context.md](../.attunic/context.md) entry that the task made stale, or that you found contradicted by the project, for the user to update or remove.
- Leave [context.md](../.attunic/context.md) and [memory.md](../.attunic/memory.md) to the user: they fill them in by hand or explicitly ask you to help; never draft or propose their content unprompted.
- Treat `workstream.md` as transient memory of this workspace: once the user consents to proceed with a workstream, create it and keep it current without asking for each edit; an accurate work-in-progress list with an interrupted workstream is a good outcome, a finished workstream with a stale one is not.
- Close a workstream by reporting the open rows of `workstream.md`'s `Found But Not Fixed` and `Deferred` tables, then delete `workstream.md` once the user agrees; the next workstream starts from a fresh skeleton.
- Never invent project knowledge: an unknown the task depends on is recorded under [question.md](../.attunic/question.md)'s `Open Questions`, not filled with a plausible guess.
- Never write to [context.md](../.attunic/context.md), [memory.md](../.attunic/memory.md), or [question.md](../.attunic/question.md) without the user's explicit confirmation of that change: these files are committed, and later sessions rely on them, so show the drafted change and ask, in a question of its own, whether to record it; a blanket approval of a plan or of several items ("yes", "proceed", "do all") does not count, and neither does approving the decision or fact itself. Without that confirmation, name the unrecorded change in your report instead.
<!-- Project-specific / Write Rules -->

## Creating Files On Demand

To create `workstream.md`, copy the skeleton in [memory-template.md](memory-template.md).
<!-- Project-specific / Creating Files On Demand -->

## Project-specific

<!-- Project-specific / Project-specific -->
