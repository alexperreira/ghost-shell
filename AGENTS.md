# AGENTS.md — ghost-shell

This repo follows Alex Perreira’s machine-wide agent defaults. Additions below are project-scoped.

## Project Info
- Owner: Alex Perreira (`@alexperreira`)
- Site/blog: alexhacks.net
- Tech stack (seed): Go, React, TypeScript, Tauri, pnpm

## Autonomy
- Read-only inspection is allowed without approval.
- Anything that changes state (file edits, installs, generators, git push) requires explicit approval unless Alex explicitly asks for that action.

## Git Policy
- Inherits machine-wide defaults (branch + push for continuity; PR required for higher-risk changes).
- Never add "Co-Authored-By: Claude" or any AI attribution to commit messages.

## Invariants (fill in)
- (Add rules that must not be broken without explicit approval.)

## Risk Zones (fill in)
- (List directories/files that require extra care.)

## Repo Quickstart (fill in as you go)

### Node / TypeScript (suggested)
- Install: `pnpm install` (or `npm install`)
- Lint: `pnpm lint`
- Test: `npm test` or `pnpm test`
- Typecheck: `pnpm typecheck`
- Dev: `pnpm dev`

### Notes
- Prefer keeping the repo on the WSL filesystem (not under `/mnt/c`) for performance.
- Update this file with concrete commands once the stack stabilizes.
