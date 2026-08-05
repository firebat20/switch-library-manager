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
| `showMessageBox` helper (JS) | ⚠️ Partial | Helper exists at `app.js:2`, but **7 call sites still use undefined `dialog.showMessageBox()`** (lines 606, 626, 639, 657, 677, 690, 745) — these will throw `ReferenceError` at runtime |
| `maximizeWindow` IPC handler (Go) | ✅ Done | Handler at `gui.go:274` |
| `electron.remote.nativeTheme` removal (JS) | ❌ Open | Lines 710, 716 still call `require('electron').remote.nativeTheme` |
| `electron.remote.getCurrentWindow()` removal (JS) | ❌ Open | Line 781 still calls `require('electron').remote.getCurrentWindow()` for window bounds tracking |
| jQuery 3.5.1 → 3.7.1 | ❌ Open | Still at v3.5.1 (`jquery-3.5.1.min.js`, header confirms) |
| Moment.js → 2.30.1 | ❌ Open | Still at old version (pre-2.29, no version header; file is 58 KB) |
| Bootstrap CSS 4.0.0 → 4.6.2 | ❌ Open | Still at v4.0.0 (header: `Bootstrap v4.0.0`) |
| `security.yml` GitHub Actions workflow | ❌ Open | No `security.yml` exists in `.github/workflows/` |

## User Review Required

> [!IMPORTANT]
> **Disabling Electron Remote Module (`EnableRemoteModule: false`)**:
> Modern Electron security best practices recommend disabling the `remote` module. To achieve this, all renderer interactions (`dialog`, `shell.showItemInFolder`, window maximize, native theme toggling) will be routed through Astilectron IPC messages to the Go backend.

> [!WARNING]
> **7 broken `dialog.showMessageBox()` calls**: The old `dialog` variable was removed but its call sites in the organize/hard-rescan flows were **not migrated** to the new `showMessageBox()` helper. These will crash at runtime. This is a **P0 bug**.

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

### Frontend UI (`resources/app/app.js` & `resources/app/app.html`) — ⚠️ PARTIALLY COMPLETE

#### [MODIFY] [app.js](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/app.js)
- ✅ Removed top-level `const { shell, dialog } = require('electron').remote`.
- ✅ Created `showMessageBox()` helper using `window.alert()`/`window.confirm()`.
- ✅ Created `openFolderPickerHelper()` using HTML5 `<input webkitdirectory>`.
- ✅ Replaced `shell.showItemInFolder` calls with `showItemInFolderHelper()` IPC calls.
- ❌ **7 `dialog.showMessageBox(null, ...)` calls NOT migrated** to the new `showMessageBox()` helper:
  - Line 606: Organize settings info dialog
  - Line 626: Organize confirmation dialog
  - Line 639: Organize success dialog
  - Line 657: Library organize info dialog
  - Line 677: Library organize confirmation dialog
  - Line 690: Library organize success dialog
  - Line 745: Hard rescan confirmation dialog
- ❌ Lines 710, 716: Still use `require('electron').remote.nativeTheme.themeSource` — should use CSS class toggle only (already partially done via `document.body.classList` manipulation).
- ❌ Line 781: Still uses `require('electron').remote.getCurrentWindow()` for window bounds tracking — should be replaced with `window.innerWidth`/`window.innerHeight` or an IPC call.

---

### Vendored Libraries (`resources/app/lib/`) — ❌ NOT STARTED

#### [MODIFY] [jQuery](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/js/jquery/jquery-3.5.1.min.js)
- ❌ Replace with jQuery 3.7.1.

#### [MODIFY] [Moment.js](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/js/moment/moment.min.js)
- ❌ Replace with Moment 2.30.1.

#### [MODIFY] [Bootstrap CSS](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/css/bootstrap/bootstrap.min.css)
- ❌ Update Bootstrap CSS to 4.6.2.

---

### Go CI / Security — ❌ NOT STARTED

#### [NEW] [.github/workflows/security.yml](file:///c:/Codes/switch-library-manager-dlo/.github/workflows/security.yml)
- ❌ Add GitHub Actions workflow running `govulncheck ./...` on push/PR to automatically flag vulnerable Go dependencies on reachable code paths.

---

## Verification Plan

### Automated Tests
- Build Go backend: `go build ./...` inside `src/`.
- Run Go tests if any exist: `go test ./...` inside `src/`.

### Manual Verification
- Launch application via bundler / executable.
- Test folder picker dialog, file click (`showItemInFolder`), organize confirmation dialog, and dark mode toggle to ensure all IPC messaging works seamlessly with `EnableRemoteModule: false`.
