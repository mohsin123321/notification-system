---
name: legacy-migration
description: >
  Before each migration run, load this skill.

  Migrate legacy code to the new carfax app preserving all observable behavior:
  exact error messages, status codes, and response shapes — even if non-standard,
  since clients may depend on them.

  If the legacy code contains bugs:
  - Migrate the buggy behavior as-is
  - Flag each bug clearly with an explanation
  - Ask for the developer's decision before proceeding

  Do NOT silently fix bugs or modernize error handling without approval.
---
