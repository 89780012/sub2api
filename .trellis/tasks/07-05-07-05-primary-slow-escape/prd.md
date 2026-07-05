# Primary account slow escape

## Goal

When a configured OpenAI group primary account becomes consistently slow or quality-degraded, routing should escape the primary account similarly to failure failover instead of continuing to force all traffic through the slow account.

## Requirements

- Apply the existing quality/sticky-escape signal to OpenAI primary account selection.
- If the current preferred primary account is quality-degraded, skip the primary hit for that request and let normal load-balanced selection choose another eligible account.
- When another account is selected after primary escape and the group allows automatic primary replacement, reuse the existing `failover_candidate` / `failover_promoted` promotion flow.
- Preserve current behavior for groups that do not allow automatic manual-primary replacement.
- Avoid changing frontend behavior or database schema.

## Acceptance Criteria

- [x] OpenAI primary selection bypasses a degraded primary account using the same quality escape thresholds already used for sticky sessions.
- [x] The bypass reason is visible in schedule trace as a primary bypass reason.
- [x] A successful replacement account can become `failover_candidate` when primary escape occurred and auto replacement is enabled.
- [x] Existing failure-based primary promotion behavior continues to pass tests.
- [x] Focused backend tests cover primary slow/quality escape.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
