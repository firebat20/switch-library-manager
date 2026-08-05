# Implementation Plan - Complete Security Hardening & Dependency Upgrades

Address the remaining follow-up security patches outlined in `SECURITY_PATCHES.md`: complete the Electron remote module migration so `EnableRemoteModule` can be set to `false`, upgrade vulnerable vendored frontend libraries in `resources/app/lib/`, and configure Go dependency vulnerability auditing.

## Status Summary

| Area | Status | Notes |
|------|--------|-------|
| `EnableRemoteModule: false` | ✅ Done | Set in `gui.go` line 130 |
| `showItemInFolder` IPC handler (Go) | ✅ Done | Handler at `gui.go:272`; `openItemInFolder()` at line 296 |
| `showItemInFolder` IPC calls (JS) | ✅ Done | `showItemInFolderHelper()` at `app.js:47` using `astilectron.sendMessage` |
| `openFolderPicker` IPC handler (Go) | ⚠️ N/A | Frontend uses `<input webkitdirectory>` fallback instead of a Go IPC handler |
| `openFolderPicker` (JS) | ✅ Done | `openFolderPickerHelper()` at `app.js:21` uses HTML5 file input |
| `showMessageBox` helper (JS) | ✅ Done | Helper at `app.js:2` returns a Promise; all former `dialog.showMessageBox()` call sites migrated (~lines 606, 623, 636, 651, 668, 681, 733) |
| `maximizeWindow` IPC handler (Go) | ✅ Done | Handler at `gui.go:274` |
| `electron.remote.nativeTheme` removal (JS) | ✅ Done | Dark mode toggled purely via `document.body.classList` + `color-scheme` meta tag |
| `electron.remote.getCurrentWindow()` removal (JS) | ✅ Done | Window bounds tracked via `window.innerWidth`/`innerHeight` in debounced `resize` listener |
| jQuery 3.5.1 → 3.7.1 | ✅ Done | `jquery-3.7.1.min.js` vendored; `app.html` reference updated |
| Moment.js → 2.30.1 | ✅ Done | Header confirms 2.30.1 |
| Bootstrap CSS 4.0.0 → 4.6.2 | ✅ Done | Header confirms 4.6.2 |
| `security.yml` GitHub Actions workflow | ✅ Done | `govulncheck -tags dev ./...` + `go vet` on push/PR/weekly schedule |
| CI bundler build fix (`dummy_assets.go`) | ✅ Done | Gated behind `//go:build dev` so it no longer collides with bundler-generated `bind*.go` (was failing GitHub Actions with "Asset redeclared") |
| CI dependency hygiene | ✅ Done | Removed `go get -u .../go-astilectron-bundler/` from setup-environment; bundler now installed from pinned go.mod version |

## User Review Required

> [!IMPORTANT]
> **Disabling Electron Remote Module (`EnableRemoteModule: false`)**:
> Modern Electron security best practices recommend disabling the `remote` module. To achieve this, all renderer interactions (`dialog`, `shell.showItemInFolder`, window maximize, native theme toggling) will be routed through Astilectron IPC messages to the Go backend.

> [!WARNING]
> **RESOLVED — 7 broken `dialog.showMessageBox()` calls**: All former `dialog.showMessageBox()` call sites in the organize/hard-rescan flows have been migrated to the Promise-based `showMessageBox()` helper. No `ReferenceError` risk remains.

> [!NOTE]
> **Frontend Dependency Updates**:
> Vendored dependencies in `src/resources/app/lib/` will be updated to safe versions:
> - **jQuery**: 3.5.1 → 3.7.1
> - **Moment.js**: 2.27.0 → 2.30.1 (fixes CVE-2022-24785 path traversal and CVE-2022-31129 ReDoS)
> - **Bootstrap CSS**: 4.0.0 → 4.6.2 (fixes CVE-2018-14040, CVE-2018-14041, CVE-2018-14042, CVE-2019-8331)

## Open Questions

- None at this stage.

## Proposed Changes

---

### Backend GUI (`gui.go`) & IPC Messaging — ✅ COMPLETE

#### [MODIFY] [gui.go](file:///c:/Codes/switch-library-manager-dlo/src/gui.go)
- ✅ Set `EnableRemoteModule: astikit.BoolPtr(false)` in `astilectron.WindowOptions`.
- ✅ Added IPC message handler `showItemInFolder` with cross-platform `openItemInFolder()`.
- ✅ Added IPC message handler `maximizeWindow`.
- ⚠️ `openFolderPicker` and `showMessageBox` were not implemented as Go IPC handlers — the frontend uses browser-native alternatives (`<input webkitdirectory>` and `window.alert`/`window.confirm`).

---

### Frontend UI (`resources/app/app.js` & `resources/app/app.html`) — ✅ COMPLETE

#### [MODIFY] [app.js](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/app.js)
- ✅ Removed top-level `const { shell, dialog } = require('electron').remote`.
- ✅ Created `showMessageBox()` helper using `window.alert()`/`window.confirm()`.
- ✅ Created `openFolderPickerHelper()` using HTML5 `<input webkitdirectory>`.
- ✅ Replaced `shell.showItemInFolder` calls with `showItemInFolderHelper()` IPC calls.
- ✅ **All 7 `dialog.showMessageBox(null, ...)` calls migrated** to the new `showMessageBox()` helper:
  - Line 606: Organize settings info dialog
  - Line 626: Organize confirmation dialog
  - Line 639: Organize success dialog
  - Line 657: Library organize info dialog
  - Line 677: Library organize confirmation dialog
  - Line 690: Library organize success dialog
  - Line 745: Hard rescan confirmation dialog
- ✅ Removed: `require('electron').remote.nativeTheme.themeSource` — should use CSS class toggle only (already partially done via `document.body.classList` manipulation).
- ✅ Removed: `require('electron').remote.getCurrentWindow()` for window bounds tracking — should be replaced with `window.innerWidth`/`window.innerHeight` or an IPC call.

---

### Vendored Libraries (`resources/app/lib/`) — ✅ COMPLETE

#### [MODIFY] [jQuery](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/js/jquery/jquery-3.5.1.min.js)
- ✅ Replaced with jQuery 3.7.1 (`jquery-3.7.1.min.js`; `app.html` updated).

#### [MODIFY] [Moment.js](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/js/moment/moment.min.js)
- ✅ Replaced with Moment 2.30.1.

#### [MODIFY] [Bootstrap CSS](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/css/bootstrap/bootstrap.min.css)
- ✅ Updated Bootstrap CSS to 4.6.2.

---

### Go CI / Security — ✅ COMPLETE

#### [NEW] [.github/workflows/security.yml](file:///c:/Codes/switch-library-manager-dlo/.github/workflows/security.yml)
- ✅ Added GitHub Actions workflow running `govulncheck -tags dev ./...` and `go vet -tags dev ./...` on push/PR to automatically flag vulnerable Go dependencies on reachable code paths.

---

## Verification Plan

### Automated Tests
- Build Go backend: `go build -tags dev ./...` inside `src/` (the `dev` tag includes `dummy_assets.go` stubs for bundler-generated symbols).
- Run Go tests if any exist: `go test -tags dev ./...` inside `src/`.

### Manual Verification
- Launch application via bundler / executable.
- Test folder picker dialog, file click (`showItemInFolder`), organize confirmation dialog, and dark mode toggle to ensure all IPC messaging works seamlessly with `EnableRemoteModule: false`.
