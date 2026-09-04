# P2 Review & Future Tasks

This document tracks lower-severity (P2) code improvements, formatting cleanups, and architectural refinements identified during the app functionality review.

---

## 1. Switch File Parsing & Metadata

### 1.1 Standardize Title ID 16-Digit Hex Formatting
* **Files:**
  - [`src/switchfs/cnmt.go:118`](file:///c:/Codes/switch-library-manager-dlo/src/switchfs/cnmt.go#L118)
  - [`src/switchfs/ncaHeader.go:79-80`](file:///c:/Codes/switch-library-manager-dlo/src/switchfs/ncaHeader.go#L79-L80)
* **Background:**
  - In `cnmt.go`: `fmt.Sprintf("0%x", titleId)` assumes `titleId` formats to 15 hex digits. If `titleId` has leading zeroes or a full 16 digits, `"0%x"` produces incorrect 17-digit Title IDs.
  - In `ncaHeader.go`: `strconv.FormatInt(int64(title_id_dec), 16)` casts `uint64` to `int64`. Any Title ID with the highest bit set becomes negative (e.g. `"-7..."`).
* **Proposed Fix:**
  Use `%016x` consistently across all Title ID formatters:
  ```go
  // cnmt.go
  TitleId: fmt.Sprintf("%016x", titleId)

  // ncaHeader.go
  result.titleId = []byte(fmt.Sprintf("%016x", title_id_dec))
  ```

### 1.2 NACP Little-Endian `SupportedLanguageFlag`
* **File:** [`src/switchfs/nacp.go:115`](file:///c:/Codes/switch-library-manager-dlo/src/switchfs/nacp.go#L115)
* **Background:**
  Nintendo Switch Horizon OS / ARM64 data structures are Little-Endian. `binary.BigEndian.Uint32` byte-swaps the flag, leading to corrupted language bitmasks.
* **Proposed Fix:**
  ```go
  supportedLanguageFlag := binary.LittleEndian.Uint32(data[offset+0x302C : offset+0x302C+0x4])
  ```

### 1.3 NACP Language 15 Collision (Brazilian Portuguese)
* **File:** [`src/switchfs/nacp.go:28-59`](file:///c:/Codes/switch-library-manager-dlo/src/switchfs/nacp.go#L28-L59)
* **Background:**
  In Switch firmware 10.0.0+, language index 15 was added for Brazilian Portuguese (`BrazilianPortuguese`). Currently, index 15 maps to `"Chinese"` in `Language.String()`, colliding with Simplified Chinese (index 14) and overwriting title entries.
* **Proposed Fix:**
  Add `BrazilianPortuguese` to the `Language` enum and update `String()`:
  ```go
  const (
      ...
      Chinese
      BrazilianPortuguese
  )

  func (l Language) String() string {
      return [...]string{
          "AmericanEnglish", "BritishEnglish", "Japanese", "French", "German",
          "LatinAmericanSpanish", "Spanish", "Italian", "Dutch", "CanadianFrench",
          "Portuguese", "Russian", "Korean", "Taiwanese", "Chinese", "BrazilianPortuguese",
      }[l]
  }
  ```

---

## 2. Title IDs & Configuration Case-Insensitivity

### 2.1 Case-Insensitive Matching in `incompleteTitleProcessor.go`
* **File:** [`src/process/incompleteTitleProcessor.go:42, 89, 137`](file:///c:/Codes/switch-library-manager-dlo/src/process/incompleteTitleProcessor.go#L42)
* **Background:**
  The `ignoreTitleIds` maps in `console.go` and `gui.go` are populated with lowercased IDs. In `incompleteTitleProcessor.go`, checking `ignoreTitleIds[switchFile.File.Metadata.TitleId]` fails if the metadata Title ID is uppercase.
* **Proposed Fix:**
  Always normalize to `strings.ToLower(...)` prior to looking up in `ignoreTitleIds`:
  ```go
  if _, ok := ignoreTitleIds[strings.ToLower(switchFile.File.Metadata.TitleId)]; ok {
      continue
  }
  ```

### 2.2 Case-Insensitive Key Names in `prod.keys`
* **File:** [`src/settings/keys.go:76-80`](file:///c:/Codes/switch-library-manager-dlo/src/settings/keys.go#L76-L80)
* **Background:**
  `InitSwitchKeys` currently stores keys using exact casing from the file. If a user's `prod.keys` uses `HEADER_KEY = ...` or mixed casing, lookups for `header_key` or `key_area_key_...` fail.
* **Proposed Fix:**
  Store normalized lowercased keys:
  ```go
  for _, key := range p.Keys() {
      value, _ := p.Get(key)
      keysInstance.keys[strings.ToLower(key)] = value
  }
  ```

---

## 3. Concurrency & Performance

### 3.1 Fine-Grained Locking in GUI Event Loop
* **File:** [`src/gui.go:141-142`](file:///c:/Codes/switch-library-manager-dlo/src/gui.go#L141-L142)
* **Background:**
  `g.state.Lock()` is held for the entirety of `handleMessage`. When long-running tasks like `updateLocalLibrary` or `organizeLibrary` run, they hold the lock for seconds or minutes while performing I/O and emitting progress events. Any IPC message arriving from the frontend blocks during this time.
* **Proposed Fix:**
  Release `g.state.Lock()` before executing long-running file operations, locking only when reading or updating `g.state.switchDB` and `g.state.localDB`.

### 3.2 Thread-Safe `settingsInstance`
* **File:** [`src/settings/settings.go:18, 96-98, 199`](file:///c:/Codes/switch-library-manager-dlo/src/settings/settings.go#L18)
* **Background:**
  `settingsInstance` is a global pointer returned directly to callers. Concurrent goroutines (e.g. background update checks, GUI resize handlers, IPC message handlers) read and write fields like `WindowWidth` and `TitlesEtag` without synchronization.
* **Proposed Fix:**
  Protect `settingsInstance` with a `sync.RWMutex`, or return a cloned copy on `ReadSettings()`.

---

## 4. Platform & Frontend Cleanups

### 4.1 Refined macOS `.app` Directory Detection
* **File:** [`src/main.go:25-31`](file:///c:/Codes/switch-library-manager-dlo/src/main.go#L25-L31)
* **Background:**
  Checking `strings.Contains(workingFolder, ".app")` can spuriously trigger on folder paths containing `.app` in their name (e.g. `/Users/dev/myapp/bin`).
* **Proposed Fix:**
  Check specifically for `.app/` boundary or path components:
  ```go
  if strings.Contains(workingFolder, ".app/") || strings.HasSuffix(workingFolder, ".app") {
      ...
  }
  ```

### 4.2 Frontend Update Check Detail Display
* **File:** [`src/resources/app/app.js:178-190`](file:///c:/Codes/switch-library-manager-dlo/src/resources/app/app.js#L178-L190)
* **Background:**
  `sendMessage("checkUpdate", ...)` passes `message.payload` to `showMessageBox()`, but `checkUpdate` returns a string boolean (`"true"` / `"false"`), making `detail` undefined.
* **Proposed Fix:**
  Have `checkUpdate` return version information or format the message box text accordingly.
