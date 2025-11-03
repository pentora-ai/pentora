## Problem

Brief description of what's broken or missing.

## Changes

- Bullet list of key changes
- Focus on WHAT changed, not file-by-file diffs

## Testing

- What tests did you add?
- How did you verify it works?
- `make test && make validate` status

## Checklist

Before submitting this PR, please ensure:

- [ ] Tests written for all new/modified code
- [ ] `make test` passing locally
- [ ] `make validate` passing locally (lint + format + spell check)
- [ ] **CLI error handling**: No `PrintError()` in main error paths; use `PrintTotalFailureSummary()` or standardized wrapper
- [ ] Commit messages follow Conventional Commits format
- [ ] PR description explains WHY (not just what changed)

## Related

Resolves #<issue-number>
