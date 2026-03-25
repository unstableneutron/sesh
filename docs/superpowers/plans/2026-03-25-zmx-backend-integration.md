# zmx Backend Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add zmx as a first-class backend for sesh list/connect flows, including deterministic backend resolution, backend-safe picker selection, and kitty-assisted handoff for connects initiated from inside zmx.

**Architecture:** Introduce an explicit backend model (`tmux` vs `zmx`) that is separate from source (`tmux`, `zmx`, `config`, `dir`, `zoxide`, `tmuxinator`). Refactor connect flow into `resolve -> execute` with deterministic precedence and source compatibility checks. Add a thin zmx runtime wrapper and a small kitty remote-control helper so zmx transitions from zmx context can be best-effort automated without using `zmx detach`.

**Tech Stack:** Go 1.24+, Cobra CLI, Testify, Bubble Tea picker, mockery v3 (`just mock` / `just test`).

---

## File Structure

1. [model/backend.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/model/backend.go) - backend enum/constants and validation helper.
2. [model/connect_opts.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/model/connect_opts.go) - connect options include backend override and internal handoff guard.
3. [model/sesh_session.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/model/sesh_session.go) and [model/connection.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/model/connection.go) - carry explicit backend identity.
4. [model/config.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/model/config.go) and [configurator/configurator.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/configurator/configurator.go) - global `default_backend` parse/validation.
5. [zmx/zmx.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/zmx/zmx.go) and [zmx/list.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/zmx/list.go) - zmx runtime wrapper (`list`, attached-state, attach, run).
6. [lister/zmx.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/lister/zmx.go), [lister/lister.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/lister/lister.go), [lister/srcs.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/lister/srcs.go), and [lister/list.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/lister/list.go) - zmx source support, optional-discovery behavior, backend-aware hide-attached.
7. [picker/picker.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/picker/picker.go) - preserve backend/source identity in chosen item.
8. [seshcli/connect.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/seshcli/connect.go), [seshcli/list.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/seshcli/list.go), and [seshcli/picker.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/seshcli/picker.go) - new flags and option wiring.
9. [connector/connect.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/connector/connect.go), [connector/connector.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/connector/connector.go), and [connector/zmx.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/connector/zmx.go) - deterministic resolution and backend execution.
10. [kitty/kitty.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/kitty/kitty.go) - best-effort remote-control handoff helper.
11. [seshcli/deps.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/seshcli/deps.go) - wire zmx and kitty dependencies.
12. [icon/icon.go](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/icon/icon.go) and [sesh.schema.json](file:///Users/thinh_nguyen/projects/personal/sesh/.worktrees/zmx-integration/sesh.schema.json) - zmx icon support and config schema update.

### Task 1: Backend Primitives and Config Validation

**Files:**
- Create: `model/backend.go`
- Modify: `model/connect_opts.go`, `model/sesh_session.go`, `model/connection.go`, `model/config.go`, `configurator/configurator.go`
- Test: `configurator/configurator_test.go`

- [ ] **Step 1: Write failing config/backend tests**

```go
func TestGetConfig_DefaultBackend(t *testing.T) {
    cfg, err := NewConfiguratorWithPath(mockOs, mockPath, mockRuntime, configFile).GetConfig()
    require.NoError(t, err)
    assert.Equal(t, model.BackendTmux, cfg.DefaultBackend)
}

func TestGetConfig_InvalidDefaultBackend(t *testing.T) {
    _, err := NewConfiguratorWithPath(mockOs, mockPath, mockRuntime, invalidFile).GetConfig()
    assert.ErrorContains(t, err, "invalid default_backend")
}
```

- [ ] **Step 2: Run targeted tests to confirm failure**

Run: `go test ./configurator -run 'TestGetConfig_(DefaultBackend|InvalidDefaultBackend)' -v`
Expected: FAIL because `default_backend` is not modeled/validated yet.

- [ ] **Step 3: Implement backend model and config defaults**

```go
type Backend string

const (
    BackendTmux Backend = "tmux"
    BackendZmx  Backend = "zmx"
)
```

Add `DefaultBackend model.Backend` in `model.Config`, add defaulting to tmux, and validate unknown values in configurator.

Also extend `model.ConnectOpts` with `Backend model.Backend` and any internal handoff guard field needed by later tasks.

- [ ] **Step 4: Re-run targeted tests**

Run: `go test ./configurator -run 'TestGetConfig_(DefaultBackend|InvalidDefaultBackend)' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add model/backend.go model/connect_opts.go model/sesh_session.go model/connection.go model/config.go configurator/configurator.go configurator/configurator_test.go
git commit -m "feat: add backend model and default_backend config validation"
```

### Task 2: Add zmx Runtime Wrapper

**Files:**
- Create: `zmx/zmx.go`, `zmx/list.go`, `zmx/list_test.go`
- Modify: `seshcli/deps.go`, `lister/lister.go`, `connector/connector.go`
- Test: `zmx/list_test.go`

- [ ] **Step 1: Write failing zmx wrapper tests**

```go
func TestListSessionsParsesClientsAndName(t *testing.T) {
    got, err := z.ListSessions()
    require.NoError(t, err)
    assert.Equal(t, "work", got[0].Name)
    assert.Equal(t, 1, got[0].Clients)
}
```

- [ ] **Step 2: Run zmx package tests and confirm failure**

Run: `go test ./zmx -v`
Expected: FAIL because package/files do not exist yet.

- [ ] **Step 3: Implement minimal zmx adapter and inject dependency**

```go
type Zmx interface {
    ListSessions() ([]*model.ZmxSession, error)
    IsAttached() bool
    CurrentSessionName() string
    Attach(name string) (string, error)
    Run(name string, args ...string) (string, error)
}
```

Wire into `BaseDeps` and pass into lister/connector constructors.

- [ ] **Step 4: Re-run zmx tests**

Run: `go test ./zmx -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add zmx/zmx.go zmx/list.go zmx/list_test.go seshcli/deps.go lister/lister.go connector/connector.go
git commit -m "feat: add zmx runtime wrapper and dependency wiring"
```

### Task 3: Add zmx to Lister and CLI Source Flags

**Files:**
- Create: `lister/zmx.go`, `lister/zmx_test.go`
- Modify: `lister/list.go`, `lister/srcs.go`, `lister/srcs_test.go`, `lister/lister.go`, `lister/caching_lister.go`, `seshcli/list.go`, `seshcli/picker.go`
- Test: `lister/srcs_test.go`, `lister/zmx_test.go`, `lister/list_test.go`, `lister/caching_lister_test.go`

- [ ] **Step 1: Add failing tests for zmx source inclusion and optional discovery**

```go
func TestSrcs_DefaultIncludesZmx(t *testing.T) {
    assert.Equal(t, []string{"tmux", "zmx", "config", "tmuxinator", "zoxide"}, srcs(ListOptions{}))
}

func TestList_OmitsZmxWhenBinaryMissing(t *testing.T) {
    sessions, err := l.List(ListOptions{})
    require.NoError(t, err)
    assert.NotContains(t, sessions.OrderedIndex, "zmx:work")
}
```

- [ ] **Step 2: Run lister tests to verify failure**

Run: `go test ./lister -run 'TestSrcs|TestList_OmitsZmxWhenBinaryMissing' -v`
Expected: FAIL.

- [ ] **Step 3: Implement zmx source strategy and CLI flags**

Add `listZmx`, `FindZmxSession`, and `GetAttachedZmxSession`. Update list/picker command flags to include `--zmx` and pass `ListOptions.Zmx`. Update caching-lister source filtering and cache refresh paths so zmx is present on cached and uncached reads.

- [ ] **Step 4: Re-run lister tests**

Run: `go test ./lister -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lister/zmx.go lister/zmx_test.go lister/list.go lister/srcs.go lister/srcs_test.go lister/lister.go lister/caching_lister.go lister/caching_lister_test.go seshcli/list.go seshcli/picker.go
git commit -m "feat: add zmx source support to list and picker"
```

### Task 4: Preserve Backend Identity in Picker Selection

**Files:**
- Modify: `picker/picker.go`, `picker/picker_test.go`, `seshcli/picker.go`, `seshcli/list.go`
- Test: `picker/picker_test.go`, `seshcli/list_test.go`

- [ ] **Step 1: Add failing picker/list duplicate-disambiguation tests**

```go
func TestUpdate_Enter_PreservesBackendIdentity(t *testing.T) {
    chosen := resultModel.ChosenSession()
    assert.Equal(t, model.BackendZmx, chosen.Backend)
    assert.Equal(t, "work", chosen.Name)
}

func TestView_DuplicateNames_ShowBackendTag(t *testing.T) {
    view := m.View()
    assert.Contains(t, view, "work [tmux]")
    assert.Contains(t, view, "work [zmx]")
}

func TestListOutput_DuplicateNames_ShowBackendTag(t *testing.T) {
    lines := formatListOutput(sessions, false)
    assert.Contains(t, lines, "work [tmux]")
    assert.Contains(t, lines, "work [zmx]")
}
```

- [ ] **Step 2: Run picker/list tests to verify failure**

Run: `go test ./picker ./seshcli -run 'Test(Update_Enter_PreservesBackendIdentity|View_DuplicateNames_ShowBackendTag|ListOutput_DuplicateNames_ShowBackendTag)' -v`
Expected: FAIL because picker currently returns only string name and duplicate names are not backend-qualified.

- [ ] **Step 3: Implement chosen session payload, duplicate backend tags, and CLI usage**

```go
func (m Model) ChosenSession() (model.SeshSession, bool)
```

Update `seshcli/picker.go` to call `Connector.Connect` with explicit backend in `ConnectOpts` when chosen session includes it. Update picker rows and `sesh list` plain output so same-name backend collisions are rendered with backend tags.

Pass source identity from picker selection to connect resolution (for example via `ConnectOpts.SourceHint` or equivalent) so `config`/`dir`/`zoxide` selections are not lost.

- [ ] **Step 4: Run picker/list tests**

Run: `go test ./picker ./seshcli -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add picker/picker.go picker/picker_test.go seshcli/picker.go seshcli/list.go seshcli/list_test.go
git commit -m "feat: disambiguate duplicate backend names in picker and list"
```

### Task 5: Refactor Connector Resolution (`resolve -> execute`)

**Files:**
- Create: `connector/connect_test.go`
- Modify: `connector/connect.go`, `connector/connector.go`, `connector/config_wildcard.go`, `lister/lister.go`, `model/connection.go`
- Test: `connector/connect_test.go`, `connector/tmux_test.go`, `connector/config_test.go`, `connector/config_wildcard_test.go`

- [ ] **Step 1: Add failing resolution-order tests**

```go
func TestConnect_PrefersContextBackendOnNameCollision(t *testing.T) {
    _, err := c.Connect("work", model.ConnectOpts{})
    require.NoError(t, err)
    mockZmx.AssertCalled(t, "Attach", "work")
}

func TestConnect_DualContextWithoutBackendErrors(t *testing.T) {
    _, err := c.Connect("work", model.ConnectOpts{})
    assert.ErrorContains(t, err, "requires --backend")
}

func TestConnect_ConfigWildcard_RespectsBackendCompatibility(t *testing.T) {
    _, err := c.Connect("~/code/project", model.ConnectOpts{Backend: model.BackendZmx})
    require.NoError(t, err)
}
```

- [ ] **Step 2: Run connector tests to confirm failure**

Run: `go test ./connector -run 'TestConnect_(PrefersContextBackendOnNameCollision|DualContextWithoutBackendErrors)' -v`
Expected: FAIL.

- [ ] **Step 3: Implement deterministic resolution algorithm**

Implement candidate backend selection, collision tie-break rules, source compatibility checks (including `config_wildcard`), and same-session zmx no-op behavior.

- [ ] **Step 4: Re-run connector tests**

Run: `go test ./connector -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add connector/connect.go connector/connect_test.go connector/connector.go connector/config_wildcard.go connector/config_wildcard_test.go lister/lister.go model/connection.go
git commit -m "refactor: implement backend-aware connect resolution"
```

### Task 6: Implement zmx Execution and Kitty-Assisted Handoff

**Files:**
- Create: `kitty/kitty.go`, `kitty/kitty_test.go`, `connector/zmx_test.go`, `connector/handoff_test.go`
- Modify: `connector/zmx.go`, `connector/tmux.go`, `connector/connector.go`, `model/connect_opts.go`, `seshcli/connect.go`, `seshcli/picker.go`, `seshcli/deps.go`
- Test: `kitty/kitty_test.go`, `connector/zmx_test.go`, `connector/handoff_test.go`, `seshcli/connect_test.go`

- [ ] **Step 1: Add failing handoff/fallback behavior-matrix tests**

```go
func TestConnectToZmx_InsideZmxWithKitty_QueuesReconnect(t *testing.T) {
    msg, err := connectToZmx(c, conn, opts)
    require.NoError(t, err)
    assert.Contains(t, msg, "handoff")
}

func TestConnectToZmx_KittyUnavailable_ReturnsManualGuidance(t *testing.T) {
    _, err := connectToZmx(c, conn, opts)
    assert.ErrorContains(t, err, "Ctrl+\\")
}

func TestConnectToTmux_InsideZmxWithKitty_QueuesReconnect(t *testing.T) {
    msg, err := connectToTmux(c, conn, opts)
    require.NoError(t, err)
    assert.Contains(t, msg, "handoff")
}

func TestConnectToZmx_SameSession_NoOp(t *testing.T) {
    msg, err := connectToZmx(c, conn, opts)
    require.NoError(t, err)
    assert.Contains(t, msg, "already in zmx session")
}

func TestConnectToZmx_InsideTmux_ReturnsManualGuidance(t *testing.T) {
    _, err := connectToZmx(c, conn, opts)
    assert.ErrorContains(t, err, "no automatic handoff")
}
```

- [ ] **Step 2: Run targeted tests to confirm failure**

Run: `go test ./connector -run 'TestConnectTo(Zmx|Tmux)|TestConnectToZmx_(SameSession_NoOp|InsideTmux_ReturnsManualGuidance)' -v`
Expected: FAIL.

- [ ] **Step 3: Implement kitty helper and zmx connect flow**

```go
type Kitty interface {
    CanRemoteControl() bool
    SendDetach(windowID string) error
    QueueCommand(windowID string, command string) error
}
```

Implement one-shot bypass marker in queued reconnect command and preserve original flags exactly (`--backend`, `--switch`, `--command`, `--tmuxinator`, `--config`, and target name) using a single reconnect-argument builder helper. Populate replay metadata in `ConnectOpts` from both `seshcli/connect.go` and `seshcli/picker.go` entrypoints. Add/validate the `--backend tmux|zmx` Cobra flag in `seshcli/connect.go` with targeted CLI tests for invalid values.

- [ ] **Step 4: Re-run targeted and package tests**

Run: `go test ./kitty ./connector ./seshcli -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kitty/kitty.go kitty/kitty_test.go connector/zmx.go connector/tmux.go connector/zmx_test.go connector/handoff_test.go connector/connector.go model/connect_opts.go seshcli/connect.go seshcli/connect_test.go seshcli/picker.go seshcli/deps.go
git commit -m "feat: add kitty-assisted handoff for zmx and tmux targets from zmx context"
```

### Task 7: Schema, Icons, Docs, and Final Verification

**Files:**
- Modify: `icon/icon.go`, `sesh.schema.json`, `README.md`, `configurator/testdata/sesh.toml`
- Test: `picker/picker_test.go`, `lister/srcs_test.go`, `configurator/configurator_test.go`

- [ ] **Step 1: Add failing assertions for zmx icon and schema field**

```go
func TestSrcIcon_Zmx(t *testing.T) {
    icn, _ := srcIcon("zmx")
    assert.NotEqual(t, "? ", icn)
}
```

- [ ] **Step 2: Run focused tests and checks**

Run: `go test ./picker ./lister ./configurator -v`
Expected: FAIL until icon/schema/documented defaults are wired.

- [ ] **Step 3: Implement docs/schema polish**

Add zmx icon glyph, document `default_backend` and `--backend/--zmx` options, and update schema for `default_backend` enum.

- [ ] **Step 4: Run full verification suite**

Run: `PATH="$(go env GOPATH)/bin:$PATH" just test`
Expected: PASS (`ok` for all packages, coverage output generated).

- [ ] **Step 5: Final commit**

```bash
git add icon/icon.go sesh.schema.json README.md configurator/testdata/sesh.toml
git add model connector lister picker seshcli zmx kitty
git commit -m "feat: integrate zmx backend across sesh connect and list flows"
```

## Execution Notes

1. Prefer @subagent-driven-development for implementation because tasks are mostly independent and reviewable in isolation.
2. Use @verification-before-completion before declaring each task done.
3. Keep generated mocks in each commit where interfaces change by running `PATH="$(go env GOPATH)/bin:$PATH" just mock`.
