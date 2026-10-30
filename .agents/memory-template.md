# Memory Template

Create a file only when its trigger arrives, by copying the fenced skeleton of its section, keeping the headings and replacing the placeholders; then apply the project-specific additions below that section's marker, if any.

## `.attunic/wip.md`

Trigger: the user consents to an audit or cleanup campaign (see [attunic-protocol.md's Write Rules](attunic-protocol.md#write-rules)).

````markdown
# WIP

State of the current audit or cleanup campaign; kept current during the work.

**Campaign**: <what is being audited or cleaned up>
**Started**: <date>   **Last updated**: <date>
**Status**: <auditing | in progress | paused | closed>

## Baseline

Recorded before the first change, verbatim:

- **Verify command**: <command>
- **Result**: <pass/fail with counts, quoted from the actual run>
- **Known-failing before we started**: <list, so nothing is blamed on this campaign>

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

<!-- Project-specific / .attunic/wip.md -->

## Project-specific

<!-- Project-specific -->
