package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/trembon/switch-library-manager/db"
	"github.com/trembon/switch-library-manager/process"
	"github.com/trembon/switch-library-manager/settings"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"go.uber.org/zap"
)

type Pair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LocalLibraryData struct {
	LibraryData []LibraryTemplateData `json:"library_data"`
	Issues      []Pair                `json:"issues"`
	NumFiles    int                   `json:"num_files"`
}

type SwitchTitle struct {
	Name        string `json:"name"`
	TitleId     string `json:"titleId"`
	Icon        string `json:"icon"`
	Region      string `json:"region"`
	ReleaseDate string `json:"release_date"`
}

type LibraryTemplateData struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Dlc     string `json:"dlc"`
	TitleId string `json:"titleId"`
	Path    string `json:"path"`
	Icon    string `json:"icon"`
	Update  int    `json:"update"`
	Region  string `json:"region"`
	Type    string `json:"type"`
}

type ProgressUpdate struct {
	Curr    int    `json:"curr"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

type State struct {
	sync.Mutex
	switchDB *db.SwitchTitlesDB
	localDB  *db.LocalSwitchFilesDB
}

type Message struct {
	Name    string `json:"name"`
	Payload string `json:"payload"`
}

type GUI struct {
	state          State
	baseFolder     string
	localDbManager *db.LocalSwitchDBManager
	sugarLogger    *zap.SugaredLogger
	// Optionally: ctx context.Context
}

var assets embed.FS

// NewGUI creates a new GUI instance but does NOT start the Wails app.
func NewGUI(baseFolder string, sugarLogger *zap.SugaredLogger) *GUI {
	return &GUI{
		state:       State{},
		baseFolder:  baseFolder,
		sugarLogger: sugarLogger,
	}
}

// Start runs the Wails app and blocks until exit.
func (g *GUI) Start() {
	// Create the menu bar
	menubar := menu.NewMenu()
	item1 := menubar.AddSubmenu("File")
	item1.AddText("Scan", nil, func(_ *menu.CallbackData) {})
	item1.AddText("Rescan", nil, func(_ *menu.CallbackData) {})
	item2 := menubar.AddSubmenu("Debug")
	item2.AddText("Open DevTools", nil, func(_ *menu.CallbackData) {})

	err := wails.Run(&options.App{
		Title:  "Switch Library Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        g.Startup,
		Menu:             menubar,
		Bind: []interface{}{
			g,
		},
	})
	if err != nil {
		g.sugarLogger.Error("Failed to start GUI:", err)
	}
}

// Startup initializes the local DB manager and switch keys.
func (g *GUI) Startup(ctx context.Context) {
	localDbManager, err := db.NewLocalSwitchDBManager(g.baseFolder)
	if err != nil {
		g.sugarLogger.Error("Failed to create local files db\n", err)
		return
	}

	settings.InitSwitchKeys(g.baseFolder)

	g.localDbManager = localDbManager
}

// OrganizeLibrary organizes the library files according to settings.
func (g *GUI) OrganizeLibrary() {
	folderToScan := settings.ReadSettings(g.baseFolder).Folder
	options := settings.ReadSettings(g.baseFolder).OrganizeOptions
	// Validate organize options
	if !process.IsOptionsValid(options) {
		zap.S().Error("the organize options in settings.json are not valid, please check that the template contains file/folder name")
		return
	}
	// Organize files by folders
	process.OrganizeByFolders(folderToScan, g.state.localDB, g.state.switchDB, g)
	// Optionally delete old update files
	if settings.ReadSettings(g.baseFolder).OrganizeOptions.DeleteOldUpdateFiles {
		process.DeleteOldUpdates(g.baseFolder, g.state.localDB, g)
	}
}

// IsKeysFileAvailable checks if the required keys file is available.
func (g *GUI) IsKeysFileAvailable() bool {
	keys, _ := settings.SwitchKeys()
	return keys != nil && keys.GetKey("header_key") != ""
}

// LoadSettings loads the application settings as JSON.
func (g *GUI) LoadSettings() string {
	return settings.ReadSettingsAsJSON(g.baseFolder)
}

// SaveSettings saves the provided settings JSON to disk.
func (g *GUI) SaveSettings(settingsJson string) error {
	s := settings.AppSettings{}
	err := json.Unmarshal([]byte(settingsJson), &s)
	if err != nil {
		return err
	}
	settings.SaveSettings(&s, g.baseFolder)
	return nil
}

// MissingGames returns a list of missing games.
func (g *GUI) MissingGames() []SwitchTitle {
	return g.getMissingGames()
}

// UpdateLocalLibrary scans and updates the local library, returning data and issues.
func (g *GUI) UpdateLocalLibrary(ignoreCache bool) (LocalLibraryData, error) {
	localDB, err := g.buildLocalDB(g.localDbManager, ignoreCache)
	if err != nil {
		g.sugarLogger.Error(err)
		return LocalLibraryData{}, err
	}
	response := LocalLibraryData{}
	libraryData := []LibraryTemplateData{}
	issues := []Pair{}
	// Iterate over all titles in the local DB
	for k, v := range localDB.TitlesMap {
		if v.BaseExist {
			version := ""
			name := ""
			// Extract version and name from metadata if available
			if v.File.Metadata.Ncap != nil {
				version = v.File.Metadata.Ncap.DisplayVersion
				name = v.File.Metadata.Ncap.TitleName["AmericanEnglish"].Title
			}
			// If updates exist, use the latest update's version
			if len(v.Updates) != 0 {
				if v.Updates[v.LatestUpdate].Metadata.Ncap != nil {
					version = v.Updates[v.LatestUpdate].Metadata.Ncap.DisplayVersion
				} else {
					version = ""
				}
			}
			// If title exists in switchDB, use its attributes
			if title, ok := g.state.switchDB.TitlesMap[k]; ok {
				if title.Attributes.Name != "" {
					name = title.Attributes.Name
				}
				libraryData = append(libraryData,
					LibraryTemplateData{
						Icon:    title.Attributes.IconUrl,
						Name:    name,
						TitleId: title.Attributes.Id,
						Update:  v.LatestUpdate,
						Version: version,
						Region:  title.Attributes.Region,
						Type:    getType(v),
						Path:    filepath.Join(v.File.ExtendedInfo.BaseFolder, v.File.ExtendedInfo.FileName),
					})
			} else {
				// Fallback to filename parsing if no metadata
				if name == "" {
					name = db.ParseTitleNameFromFileName(v.File.ExtendedInfo.FileName)
				}
				libraryData = append(libraryData,
					LibraryTemplateData{
						Name:    name,
						Update:  v.LatestUpdate,
						Version: version,
						Type:    getType(v),
						TitleId: v.File.Metadata.TitleId,
						Path:    v.File.ExtendedInfo.FileName,
					})
			}
		} else {
			// Add issues for missing base files
			for _, update := range v.Updates {
				issues = append(issues, Pair{Key: filepath.Join(update.ExtendedInfo.BaseFolder, update.ExtendedInfo.FileName), Value: "base file is missing"})
			}
			for _, dlc := range v.Dlc {
				issues = append(issues, Pair{Key: filepath.Join(dlc.ExtendedInfo.BaseFolder, dlc.ExtendedInfo.FileName), Value: "base file is missing"})
			}
		}
	}
	// Add skipped files as issues
	for k, v := range localDB.Skipped {
		issues = append(issues, Pair{Key: filepath.Join(k.BaseFolder, k.FileName), Value: v.ReasonText})
	}
	response.LibraryData = libraryData
	response.NumFiles = localDB.NumFiles
	response.Issues = issues
	return response, nil
}

// UpdateDB updates the Switch titles database if not already loaded.
func (g *GUI) UpdateDB() error {
	if g.state.switchDB == nil {
		switchDb, err := g.buildSwitchDb()
		if err != nil {
			g.sugarLogger.Error(err)
			return err
		}
		g.state.switchDB = switchDb
	}
	return nil
}

// MissingUpdates returns a JSON string of missing updates.
func (g *GUI) MissingUpdates() string {
	return g.getMissingUpdates()
}

// MissingDlc returns a JSON string of missing DLCs.
func (g *GUI) MissingDlc() string {
	return g.getMissingDLC()
}

// CheckUpdate checks for application updates.
func (g *GUI) CheckUpdate() (bool, error) {
	newUpdate, err := settings.CheckForUpdates()
	if err != nil {
		g.sugarLogger.Error(err)
		return false, err
	}
	return newUpdate, nil
}

// getType returns the type of the game file (split, multi-content, or extension).
func getType(gameFile *db.SwitchGameFiles) string {
	if gameFile.IsSplit {
		return "split"
	}
	if gameFile.MultiContent {
		return "multi-content"
	}
	ext := filepath.Ext(gameFile.File.ExtendedInfo.FileName)
	if len(ext) > 1 {
		return ext[1:]
	}
	return ""
}

// getMissingDLC scans for missing DLCs and returns them as a JSON string.
func (g *GUI) getMissingDLC() string {
	settingsObj := settings.ReadSettings(g.baseFolder)
	ignoreIds := map[string]struct{}{}
	for _, id := range settingsObj.IgnoreDLCTitleIds {
		ignoreIds[strings.ToLower(id)] = struct{}{}
	}
	missingDLC := process.ScanForMissingDLC(g.state.localDB.TitlesMap, g.state.switchDB.TitlesMap, ignoreIds)
	values := make([]process.IncompleteTitle, len(missingDLC))
	i := 0
	for _, missingUpdate := range missingDLC {
		values[i] = missingUpdate
		i++
	}

	msg, _ := json.Marshal(values)
	return string(msg)
}

// getMissingUpdates scans for missing updates and returns them as a JSON string.
func (g *GUI) getMissingUpdates() string {
	settingsObj := settings.ReadSettings(g.baseFolder)
	ignoreIds := map[string]struct{}{}
	for _, id := range settingsObj.IgnoreUpdateTitleIds {
		ignoreIds[strings.ToLower(id)] = struct{}{}
	}
	missingUpdates := process.ScanForMissingUpdates(g.state.localDB.TitlesMap, g.state.switchDB.TitlesMap, ignoreIds, settingsObj.IgnoreDLCUpdates)
	values := make([]process.IncompleteTitle, len(missingUpdates))
	i := 0
	for _, missingUpdate := range missingUpdates {
		values[i] = missingUpdate
		i++
	}

	msg, _ := json.Marshal(values)
	return string(msg)
}

// buildSwitchDb downloads and builds the Switch titles database.
func (g *GUI) buildSwitchDb() (*db.SwitchTitlesDB, error) {
	settingsObj := settings.ReadSettings(g.baseFolder)
	// Step 1: Download titles.json
	g.UpdateProgress(1, 4, "Downloading titles.json")
	filename := filepath.Join(g.baseFolder, settings.TITLE_JSON_FILENAME)
	titleFile, titlesEtag, err := db.LoadAndUpdateFile(settingsObj.TitlesJsonUrl, filename, settingsObj.TitlesEtag)
	if err != nil {
		return nil, errors.New("failed to download switch titles [reason:" + err.Error() + "]")
	}
	settingsObj.TitlesEtag = titlesEtag

	// Step 2: Download versions.json
	g.UpdateProgress(2, 4, "Downloading versions.json")
	filename = filepath.Join(g.baseFolder, settings.VERSIONS_JSON_FILENAME)
	versionsFile, versionsEtag, err := db.LoadAndUpdateFile(settingsObj.VersionsJsonUrl, filename, settingsObj.VersionsEtag)
	if err != nil {
		return nil, errors.New("failed to download switch updates [reason:" + err.Error() + "]")
	}
	settingsObj.VersionsEtag = versionsEtag

	// Step 3: Save updated settings
	settings.SaveSettings(settingsObj, g.baseFolder)

	// Step 4: Process titles and updates
	g.UpdateProgress(3, 4, "Processing switch titles and updates ...")
	switchTitleDB, err := db.CreateSwitchTitleDB(titleFile, versionsFile)
	g.UpdateProgress(4, 4, "Finishing up...")
	return switchTitleDB, err
}

// buildLocalDB scans folders and builds the local files database.
func (g *GUI) buildLocalDB(localDbManager *db.LocalSwitchDBManager, ignoreCache bool) (*db.LocalSwitchFilesDB, error) {
	folderToScan := settings.ReadSettings(g.baseFolder).Folder
	recursiveMode := settings.ReadSettings(g.baseFolder).ScanRecursively

	scanFolders := settings.ReadSettings(g.baseFolder).ScanFolders
	scanFolders = append(scanFolders, folderToScan)
	localDB, err := localDbManager.CreateLocalSwitchFilesDB(scanFolders, g, recursiveMode, ignoreCache)
	g.state.localDB = localDB
	return localDB, err
}

// getMissingGames returns a list of SwitchTitle for games missing from the local library.
func (g *GUI) getMissingGames() []SwitchTitle {
	var result []SwitchTitle
	for k, v := range g.state.switchDB.TitlesMap {
		if _, ok := g.state.localDB.TitlesMap[k]; ok {
			continue
		}
		if v.Attributes.Name == "" || v.Attributes.Id == "" {
			continue
		}

		options := settings.ReadSettings(g.baseFolder)
		if options.HideDemoGames && v.Attributes.IsDemo {
			continue
		}

		result = append(result, SwitchTitle{
			TitleId:     v.Attributes.Id,
			Name:        v.Attributes.Name,
			Icon:        v.Attributes.BannerUrl,
			Region:      v.Attributes.Region,
			ReleaseDate: v.Attributes.ParsedReleaseDate,
		})
	}
	return result

}

// UpdateProgress sends a progress update message (for compatibility, may be adapted for Wails).
func (g *GUI) UpdateProgress(curr int, total int, message string) {
	progressMessage := ProgressUpdate{curr, total, message}
	g.sugarLogger.Debugf("%v (%v/%v)", message, curr, total)
	_, err := json.Marshal(progressMessage)
	if err != nil {
		g.sugarLogger.Error(err)
		return
	}

	// To send progress to the frontend, use Wails events (uncomment if needed):
	// import "github.com/wailsapp/wails/v2/pkg/runtime"
	// runtime.EventsEmit(g.ctx, "updateProgress", progressMessage)
}
