# Memory Template

Create a file only when its trigger arrives, by copying the fenced skeleton of its section, keeping the headings and replacing the placeholders; then apply the project-owned content below that section's marker, if any.

## `.attunic/workstream.md`

Trigger: the user consents to an audit or cleanup workstream (see [attunic-protocol.md's Write Rules](attunic-protocol.md#write-rules)).

````markdown
# Workstream

State of the current audit or cleanup workstream; kept current during the work.

**Workstream**: <what is being audited or cleaned up>
**Started**: <date>   **Last updated**: <date>
**Status**: <auditing | in progress | paused | closed>

## Baseline

Recorded before the first change, verbatim:

- **Verify command**: <command>
- **Result**: <pass/fail with counts, quoted from the actual run>
- **Known-failing before we started**: <list, so nothing is blamed on this workstream>

## Batches

One batch = one intent; never mix a mechanical sweep with structural edits in the same diff.

- [ ] 1. <module or smell family> - <what changes> - commit: <sha, if the user asked for one>

## Found But Not Fixed

Bugs and risks discovered while cleaning up: logged, not fixed, because a cleanup batch must not change behavior.

| What | Where | Severity | Note |
|------|-------|----------|------|

## Deferred

Deliberately out of scope, with the reason.

| What | Why deferred | Revisit when |
|------|--------------|--------------|
````

<!-- Project-specific / .attunic/workstream.md -->

## Project-specific

<!-- Project-specific / Project-specific -->
