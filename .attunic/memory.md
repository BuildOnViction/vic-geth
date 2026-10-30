# Memory

The reading and writing rules for this file and the rest of `.attunic/` are defined in [attunic-protocol.md](../.agents/attunic-protocol.md).

Append-only log of confirmed exceptions and deferrals, with the reasons a later session should not have to rediscover. It is a record, not a rule set: only [context.md's Confirmed Facts](context.md#confirmed-facts) are binding. Newest entry last. Supersede an old entry by adding a new one that references it; never edit the past. Append an entry only after the user explicitly confirms that entry (see [attunic-protocol.md's Write Rules](../.agents/attunic-protocol.md#write-rules)).

## Decisions

Log each confirmed exception or deferral as a decision; the fenced content below is the template for a new decision.

```
### <YYYY-MM-DD> - <short title>

**Decision**: <what was decided, in one sentence>

**Context**: <what forced the choice: a requirement, a constraint, a failure>

**Alternatives**: <what else was considered, and why each lost>

**Consequences**: <what this makes easy, and what it makes hard or more expensive>

**Revisit if**: <the condition that would change the answer, or "n/a">
```

<!-- Project-specific / Decisions -->

## Open Questions

Things the user needs to answer, or that nobody currently knows; better admitted than guessed.

Add each question as one bullet: the question, and who or what could answer it. Close a question by appending its answer to that bullet; the question itself stays.
<!-- Project-specific / Open Questions -->

## Project-specific

<!-- Project-specific -->
