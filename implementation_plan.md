# Implementation Plan - Complete Security Hardening & Dependency Upgrades

Address the remaining follow-up security patches outlined in `SECURITY_PATCHES.md`: complete the Electron remote module migration so `EnableRemoteModule` can be set to `false`, upgrade vulnerable vendored frontend libraries in `resources/app/lib/`, and configure Go dependency vulnerability auditing.

## User Review Required

> [!IMPORTANT]
> **Disabling Electron Remote Module (`EnableRemoteModule: false`)**:
> Modern Electron security best practices recommend disabling the `remote` module. To achieve this, all renderer interactions (`dialog`, `shell.showItemInFolder`, window maximize, native theme toggling) will be routed through Astilectron IPC messages to the Go backend.

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

### Backend GUI (`gui.go`) & IPC Messaging

#### [MODIFY] [gui.go](file:///c:/Codes/switch-library-manager-dlo/src/gui.go)
- Set `EnableRemoteModule: astikit.BoolPtr(false)` in `astilectron.WindowOptions`.
- Add IPC message handlers in `handleMessage`:
  - `openFolderPicker`: Opens system directory picker and returns selected folder path.
  - `showItemInFolder`: Opens OS file manager highlighting the requested file (`explorer /select,` on Windows, `open -R` on macOS, `dbus` / `xdg-open` on Linux).
  - `showMessageBox`: Displays native system message box or confirmation dialog.
  - `maximizeWindow` / `getWindowBounds`: Manages window state without requiring renderer access to `electron.remote`.

---

### Frontend UI (`resources/app/app.js` & `resources/app/app.html`)

#### [MODIFY] [app.js](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/app.js)
- Remove `const { shell, dialog } = require('electron').remote`.
- Replace direct `dialog.showOpenDialog` and `dialog.showMessageBox` calls with `astilectron.sendMessage` calls to the Go backend.
- Replace `shell.showItemInFolder` cell clicks with `astilectron.sendMessage("showItemInFolder", path)`.
- Replace `electron.remote.getCurrentWindow()` calls with IPC calls or local state management.
- Replace `electron.remote.nativeTheme` calls with standard CSS dark mode class toggle.

---

### Vendored Libraries (`resources/app/lib/`)

#### [MODIFY] [jQuery](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/js/jquery/jquery-3.5.1.min.js)
- Replace with jQuery 3.7.1.

#### [MODIFY] [Moment.js](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/js/moment/moment.min.js)
- Replace with Moment 2.30.1.

#### [MODIFY] [Bootstrap CSS](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/lib/css/toggle-bootstrap.min.css)
- Update Bootstrap CSS files to 4.6.2.

---

### Go CI / Security

#### [NEW] [.github/workflows/security.yml](file:///c:/Codes/switch-library-manager-dlo/.github/workflows/security.yml)
- Add GitHub Actions workflow running `govulncheck ./...` on push/PR to automatically flag vulnerable Go dependencies on reachable code paths.

---

## Verification Plan

### Automated Tests
- Build Go backend: `go build ./...` inside `src/`.
- Run Go tests if any exist: `go test ./...` inside `src/`.

### Manual Verification
- Launch application via bundler / executable.
- Test folder picker dialog, file click (`showItemInFolder`), organize confirmation dialog, and dark mode toggle to ensure all IPC messaging works seamlessly with `EnableRemoteModule: false`.
