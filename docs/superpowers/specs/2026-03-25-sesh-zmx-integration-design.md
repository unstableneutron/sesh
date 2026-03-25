# Sesh zmx Integration Design

Date: 2026-03-25
Status: Approved (design)

## Summary

Add zmx as a first-class session backend in sesh while preserving tmux behavior. Keep backend behavior honest: use native tmux switching where available, and use a kitty-assisted handoff for zmx transitions initiated from inside zmx when possible. Fall back to explicit manual guidance when automation is not possible.

## Goals

1. Add zmx as a supported backend for listing and connecting.
2. Keep tmux behavior unchanged for existing users.
3. Support backend-aware session resolution with deterministic tie-breaking.
4. Provide practical switch-like behavior for zmx sessions via kitty remote control when available.
5. Add global `default_backend` config support.

## Non-Goals

1. Full tmux feature parity on zmx.
2. Native zmx session switching (zmx does not provide this today).
3. Automatic `zmx detach` command execution (it detaches all clients).
4. Per-session backend overrides in config (deferred).
5. A generalized plugin framework for arbitrary session backends.

## Current Constraints

1. zmx attach refuses when already inside a zmx session (`ZMX_SESSION` set).
2. zmx has no native `switch` command.
3. zmx `detach` command detaches all clients and is unsafe to use for targeted switching.
4. sesh currently routes connect behavior mostly through tmux-specific execution paths.

## Proposed Architecture

### Session Backend Model

Introduce an explicit backend dimension separate from discovery source:

1. `src` means where a session came from (`tmux`, `zmx`, `config`, `zoxide`, `dir`, `tmuxinator`).
2. `backend` means which runtime executes the connect (`tmux` or `zmx`).

This split avoids overloading `src` and allows config/dir/zoxide sessions to resolve to either backend.

### Source and Backend Compatibility

Allowed source-to-backend combinations for MVP:

1. `src=tmux` -> `backend=tmux` only.
2. `src=zmx` -> `backend=zmx` only.
3. `src=tmuxinator` -> `backend=tmux` only.
4. `src=config` -> `backend=tmux|zmx`.
5. `src=dir` -> `backend=tmux|zmx`.
6. `src=zoxide` -> `backend=tmux|zmx`.

If resolution produces an invalid source/backend combination, sesh must return an explicit error.

### zmx Runtime Wrapper

Add a thin `zmx` package wrapper similar to `tmux` wrappers used today.

Expected capabilities:

1. List sessions.
2. Detect attached state (`ZMX_SESSION`).
3. Read current zmx session name.
4. Attach (create-or-attach semantics via `zmx attach`).
5. Optional command execution support via `zmx run` where needed.

### Connector Refactor

Refactor connect flow into two stages:

1. Resolve target session and backend.
2. Execute backend-specific connect strategy.

Do not route connect execution by `src`. Route by resolved `backend`.

## Backend Resolution Rules

For `sesh connect <name>`, resolution is deterministic and ordered:

1. Detect current context (`TMUX`, `ZMX_SESSION`).
2. If both tmux and zmx context are present and `--backend` is not set, return an ambiguity error that requires `--backend`.
3. Determine candidate backends.
4. If `--backend` is set, only that backend is a candidate.
5. If `--backend` is not set, both backends are candidates.
6. Search existing runtime sessions by name across candidate backends.
7. If exactly one backend has a match, select it.
8. If both backends have matches, apply tie-break order: active context backend, then `default_backend`, then tmux.
9. If neither backend has a runtime match, choose creation backend with the same precedence: `--backend`, active context backend, `default_backend`, tmux.
10. Validate source/backend compatibility before execute.
11. If resolved backend is zmx and `ZMX_SESSION` already matches target session name, return no-op success and skip handoff.

Picker-selected connects use the same rules, but any picker-provided backend identity constrains candidate backends at step 3.

### Source Candidate Selection

After backend is resolved, source selection for same-name candidates is deterministic:

1. Prefer runtime source candidates in the resolved backend (`src=tmux` for tmux backend, `src=zmx` for zmx backend).
2. If no runtime candidate exists, use non-runtime source precedence: `tmuxinator` (tmux only), `config`, `dir`, then `zoxide`.
3. Skip any candidate that is incompatible with the resolved backend.
4. If multiple candidates remain within the same precedence tier, return an explicit ambiguity error.

## CLI and Config Changes

### `connect`

1. Add `--backend tmux|zmx`.
2. Keep `--switch`; tmux uses native switch behavior.
3. For zmx paths, `--switch` is a strict alias of normal connect semantics and does not change behavior.

### `list` and picker

1. Add zmx to source listing and picker discovery.
2. Include zmx entries in default source set.
3. Keep source flags consistent with existing command style.
4. Treat optional backend discovery as best-effort: if zmx is not installed, default list/picker output omits zmx sessions without failing.
5. Preserve backend identity in entries and selection payload so duplicate names are unambiguous.

### Config

Add global config key:

1. `default_backend = "tmux" | "zmx"`

Per-session backend config is deferred.

### Picker and Duplicate Identity

Duplicate same-name sessions across backends are supported in MVP.

1. Picker/list display must distinguish entries by backend (for example, `foo [tmux]` and `foo [zmx]`).
2. Picker selection must pass explicit backend identity into connect resolution.
3. Selecting `foo [zmx]` must not silently resolve to tmux `foo`.
4. Backend-agnostic sources (`config`, `dir`, `zoxide`) appear once per source entry, not duplicated per backend.
5. Backend-agnostic picker selections pass source identity and name, then backend is resolved via the connect algorithm.

## zmx Switching Behavior

### Baseline

zmx does not support native in-session switch. sesh must emulate handoff.

### Kitty-Assisted Handoff (Default)

When all conditions are true:

1. current context is inside zmx,
2. target backend is tmux or zmx,
3. kitty remote control env is present (`KITTY_WINDOW_ID`, `KITTY_LISTEN_ON`, `KITTY_PUBLIC_KEY`),

sesh performs best-effort handoff:

1. send `ctrl+backslash` to current kitty window,
2. queue a follow-up `sesh connect ...` command in the same window,
3. return after issuing handoff command.

Handoff success and failure criteria for MVP:

1. Success means both kitty remote-control commands (`send-key` and follow-up `send-text`) return success.
2. Failure means missing kitty prerequisites, missing `kitten`, or either remote-control command returning a failure.
3. MVP does not require synchronous confirmation that detach completed before returning.

### Loop Protection

Queued reconnect command must include an internal one-shot marker to prevent recursion if handoff path is re-entered.

The queued reconnect command must replay the original connect intent exactly (target name, backend choice, and relevant flags), then clear the one-shot marker.

### Manual Fallback

If kitty capability is unavailable or handoff fails:

1. show explicit guidance to press `Ctrl+\` in current session,
2. show exact command to rerun.

## Behavior Matrix

1. inside tmux, target tmux: native tmux switch/attach behavior.
2. inside tmux, target zmx: no automatic handoff in MVP; return actionable manual guidance.
3. inside zmx, target zmx same session: return no-op success.
4. inside zmx, target zmx different session: use kitty-assisted handoff by default; fallback to manual detach guidance.
5. inside zmx, target tmux: use kitty-assisted handoff by default; fallback to manual detach guidance.
6. outside both: normal backend attach/create behavior.

## Error Handling and Messages

Messages must be explicit and actionable.

Examples:

1. Missing kitty capability for zmx handoff: explain why automatic handoff is unavailable and provide exact follow-up command.
2. Invalid backend flag value: show allowed values.
3. zmx binary missing when backend resolved to zmx: clear dependency error.
4. tmuxinator with zmx backend: explicit incompatibility error.
5. dual-context ambiguity (`TMUX` and `ZMX_SESSION` both set) without `--backend`: explicit disambiguation error.

## Testing Strategy

1. Unit tests for backend resolution ordering.
2. Unit tests for ambiguous name tie-breaking.
3. Unit tests for kitty handoff path decision logic.
4. Unit tests for loop-protection marker handling.
5. Integration tests for zmx list/connect wrappers (with command stubs/mocks).
6. Regression tests for existing tmux connect/list behavior.

## Rollout Plan

### Phase 1 (MVP)

1. Add backend model and config `default_backend`.
2. Add zmx list and connect support.
3. Add backend-aware resolution and execution routing.
4. Implement backend-qualified picker/list identity for duplicate names.
5. Implement kitty-assisted zmx handoff with fallback guidance.
6. Keep per-session backend config out of scope.

### Phase 2 (Follow-up)

1. Per-session backend config.
2. Enhanced list/picker grouping and richer backend visuals.
3. Expand preview behavior where backend-appropriate.

## Risks and Mitigations

1. Risk: fragile terminal automation around remote-control handoff.
   Mitigation: best-effort path with strict fallback and no destructive detach-all command.
2. Risk: accidental recursion in queued reconnect.
   Mitigation: one-shot internal marker guard.
3. Risk: behavior confusion between native and emulated switching.
   Mitigation: clear messaging and backend-aware logs/messages.

## Open Follow-ups

1. Decide whether to expose an opt-out flag for kitty-assisted handoff.
2. Decide long-term UX for duplicate same-name sessions across backends in picker output.
3. Revisit behavior once zmx introduces native switch semantics.
