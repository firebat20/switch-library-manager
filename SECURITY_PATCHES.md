# Security & bug-fix patches — switch-library-manager

This document lists every change applied to the source tree and the follow-up
items that require a network connection / manual dependency updates (which could
not be applied in the offline patching environment).

## Applied code patches

### 1. Stored/DOM XSS in the Electron UI  (HIGH)  — `resources/app/app.js`
The Tabulator table formatters built HTML by string interpolation from
untrusted data (title names, DLC names, and icon URLs come from the remotely
downloaded `titles.json`/`versions.json` — whose URLs are user-editable — and
from NACP metadata inside scanned game files). A crafted value such as
`<img src=x onerror=...>` executed in the renderer. Because the window runs with
the Electron **remote module enabled**, that DOM XSS was a path to code
execution.

Fixes:
- Added `escapeHtml()` and `sanitizeImageUrl()` helpers.
- `fluentTitleFormatter`: title text is escaped; the image URL is restricted to
  `http(s):`/`data:image` and then escaped.
- `fluentFileFormatter`: file and directory names are escaped (they land in both
  element text and `title=""` attributes).
- "Issue" column: value is escaped *before* the known literal markers are turned
  into `<br/>`/`<strong>` markup.
- "Missing DLC" column: each DLC name is escaped; also fixed an implicit-global
  `value` and a missing null guard.

Note: the jsrender templates in `app.html` use `{{:...}}`, which auto
HTML-encodes, so those were already safe. No `{{>...}}` (raw) outputs exist.

### 2. Electron remote module  (HIGH — partially addressed)  — `gui.go`
`EnableRemoteModule: true` bridges renderer JS to Node/main-process APIs.
The practical exploit (issue #1) is now closed, but the remote module remains a
latent risk. Left enabled with a prominent `SECURITY NOTE` because the renderer
still calls `electron.remote` for dialogs, `shell`, window control and
`nativeTheme`. **Recommended follow-up:** route those through astilectron
messages to Go and set `EnableRemoteModule` to `false` (ideally also enabling
`contextIsolation` and disabling `nodeIntegration` if the astilectron fork
supports them).

### 3. HTTP client hardening  (MEDIUM)  — `db/utils.go`, `settings/settings.go`
- `downloadBytesFromUrl`: added a full-request `Timeout` (60s) plus
  `TLSHandshakeTimeout`/`ResponseHeaderTimeout`; previously only a 3s *dial*
  timeout existed, so a server that stalled the body hung the app forever.
- Bounded the response body with `io.LimitReader` (256 MB cap) so a hostile/
  misconfigured endpoint can't force an unbounded allocation.
- `CheckForUpdates`: replaced the bare `http.Get` (no timeout at all) with a
  client that has a 30s timeout, a status-code check, and a 1 MB bounded read.

### 4. Nil-pointer panics  (MEDIUM)  — `gui.go`, `process/organizefolderStructure.go`
- `gui.go` `updateLocalLibrary`: `v.Updates[v.LatestUpdate].Metadata.Ncap`
  panicked when `LatestUpdate` wasn't a key (zero-value `SwitchFileInfo` has a
  nil `Metadata`). Now looked up with the comma-ok form and nil-checked.
- `organizefolderStructure.go` updates loop: `updateInfo.Metadata.Ncap` now
  checks `Metadata != nil` first.

### 5. `DeleteOldUpdates` operated on the wrong directory  (MEDIUM)
It received `baseFolder` (the app's install directory) and then ran
`deleteEmptyFolders` against it, so empty-folder pruning could delete
directories next to the executable. Both call sites (`gui.go`, `console.go`)
now pass the library `folderToScan`, matching `OrganizeByFolders`.

### 6. World-writable directory creation  (LOW)
`os.Mkdir(path, os.ModePerm)` (0777) → `os.MkdirAll(path, 0755)` in
`process/organizefolderStructure.go` and `console.go`. `MkdirAll` also creates
missing parents.

### 7. Silent settings-save failures + permissions  (LOW)  — `settings/settings.go`
`SaveSettings` ignored both the marshal and write errors. Now both are logged,
and the settings file is written `0600` (it references the prod.keys path and
download URLs; no need for it to be world-readable).

---

## Follow-up items requiring network access (NOT applied here)

These are vendored, minified third-party bundles under `resources/app/lib/`.
They should be replaced with upstream builds — they can't be safely hand-edited.

- **Bootstrap 4.0.0 → latest 4.6.x (or 5.x).** Affected by XSS issues
  CVE-2018-14040/14041/14042 and CVE-2019-8331.
- **moment 2.27.0 → ≥2.29.4.** CVE-2022-24785 (path traversal in locale
  loading — relevant under Electron/Node) and CVE-2022-31129 (ReDoS). moment is
  only used indirectly by Tabulator's `sorter:"date"`; if you upgrade Tabulator
  to a version that uses luxon, moment can be dropped entirely.
- **Tabulator 4.7.1 → 5.x/6.x.** Several years of fixes; also removes the moment
  dependency.
- **jQuery 3.5.1 → 3.7.x** (defense in depth; 3.5.1 has no critical known CVE
  but is old).

### Go dependencies
- Run `govulncheck` in CI:
  `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`
  (couldn't be run offline here). It reports only vulnerabilities on reachable
  code paths, which is more reliable than comparing version strings.
- Five dependencies are personal forks pinned by untagged pseudo-versions
  (`firebat20/go-astilectron`, `-bootstrap`, `-bundler`, `go-bindata`). Consider
  `go mod vendor` so builds are reproducible and don't silently pick up changes,
  and verify the **Electron version** the astilectron fork downloads/runs — an
  outdated Electron is the single biggest residual risk and matters more than
  any pure-Go library version, especially alongside issue #2.
- `go-bindata` is unmaintained upstream (build-time only; lower concern).

### Build note
`Asset`, `AssetDir`, and `RestoreAssets` are generated by `astilectron-bundler`
(go-bindata) and are not in the tree, so `go build` alone will fail; the bundler
step is required to produce a runnable binary.
