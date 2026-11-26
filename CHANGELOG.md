# Changelog (AVIBE)

Dated delivery notes live in `docs/handoff/`; this file tracks user-visible
behaviour changes per release. No git tags exist yet; cut the first tag
(`v0.1.0`) at launch readiness and move items below into it.

## Unreleased

- License: MIT (`LICENSE`); third-party notices in `THIRD_PARTY_NOTICES.md`.
- Prod hardening: TLS-only DB DSN in prod, CI shellcheck + image builds,
  rollout `--dry-run` + typed approval, compose healthchecks.
- Web: i18n routing/SSR/a11y fixes, `useToast` request scoping, lint gate.
- Mobile: API/store test coverage (59 tests), release-host gate.
- Generator: pytest/ruff CI job, spectral fingerprint screening.
