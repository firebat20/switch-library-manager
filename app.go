package main

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/trembon/switch-library-manager/db"
	"github.com/trembon/switch-library-manager/process"
	"github.com/trembon/switch-library-manager/settings"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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

type App struct {
	ctx            context.Context
	state          State
	baseFolder     string
	localDbManager *db.LocalSwitchDBManager
	sugarLogger    *zap.SugaredLogger
}

func NewApp(baseFolder string, sugarLogger *zap.SugaredLogger) *App {
	return &App{
		state:       State{},
		baseFolder:  baseFolder,
		sugarLogger: sugarLogger,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	localDbManager, err := db.NewLocalSwitchDBManager(a.baseFolder)
	if err != nil {
		a.sugarLogger.Error("Failed to create local files db\n", err)
		return
	}

	settings.InitSwitchKeys(a.baseFolder)
	a.localDbManager = localDbManager
}

func (a *App) shutdown(ctx context.Context) {
	if a.localDbManager != nil {
		a.localDbManager.Close()
	}
}

func (a *App) OrganizeLibrary() {
	folderToScan := settings.ReadSettings(a.baseFolder).Folder
	options := settings.ReadSettings(a.baseFolder).OrganizeOptions
	if !process.IsOptionsValid(options) {
		zap.S().Error("the organize options in settings.json are not valid, please check that the template contains file/folder name")
		runtime.EventsEmit(a.ctx, "error", "the organize options in settings.json are not valid, please check that the template contains file/folder name")
		return
	}
	if settings.ReadSettings(a.baseFolder).OrganizeOptions.DeleteOldUpdateFiles {
		process.DeleteOldUpdates(a.baseFolder, a.state.localDB, a)
	}
	process.OrganizeByFolders(folderToScan, a.state.localDB, a.state.switchDB, a)
}

func (a *App) IsKeysFileAvailable() bool {
	keys, _ := settings.SwitchKeys()
	return keys != nil && keys.GetKey("header_key") != ""
}

func (a *App) LoadSettings() string {
	return settings.ReadSettingsAsJSON(a.baseFolder)
}

func (a *App) SaveSettings(settingsJson string) error {
	s := settings.AppSettings{}
	err := json.Unmarshal([]byte(settingsJson), &s)
	if err != nil {
		return err
	}
	settings.SaveSettings(&s, a.baseFolder)
	return nil
}

func (a *App) GetMissingGames() []SwitchTitle {
	var result []SwitchTitle
	a.state.Lock()
	defer a.state.Unlock()

	if a.state.switchDB == nil || a.state.localDB == nil {
		return result
	}

	for k, v := range a.state.switchDB.TitlesMap {
		if _, ok := a.state.localDB.TitlesMap[k]; ok {
			continue
		}
		if v.Attributes.Name == "" || v.Attributes.Id == "" {
			continue
		}

		options := settings.ReadSettings(a.baseFolder)
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

func (a *App) UpdateLocalLibrary(ignoreCache bool) (LocalLibraryData, error) {
	a.state.Lock()
	defer a.state.Unlock()

	localDB, err := a.buildLocalDB(a.localDbManager, ignoreCache)
	if err != nil {
		a.sugarLogger.Error(err)
		return LocalLibraryData{}, err
	}

	response := LocalLibraryData{}
	libraryData := []LibraryTemplateData{}
	issues := []Pair{}

	for k, v := range localDB.TitlesMap {
		if v.BaseExist {
			version := ""
			name := ""
			if v.File.Metadata.Ncap != nil {
				version = v.File.Metadata.Ncap.DisplayVersion
				name = v.File.Metadata.Ncap.TitleName["AmericanEnglish"].Title
			}

			if len(v.Updates) != 0 {
				if v.Updates[v.LatestUpdate].Metadata.Ncap != nil {
					version = v.Updates[v.LatestUpdate].Metadata.Ncap.DisplayVersion
				} else {
					version = ""
				}
			}
			if a.state.switchDB != nil {
				if title, ok := a.state.switchDB.TitlesMap[k]; ok {
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
				// Fallback if switchDB is not loaded
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
			for _, update := range v.Updates {
				issues = append(issues, Pair{Key: filepath.Join(update.ExtendedInfo.BaseFolder, update.ExtendedInfo.FileName), Value: "base file is missing"})
			}
			for _, dlc := range v.Dlc {
				issues = append(issues, Pair{Key: filepath.Join(dlc.ExtendedInfo.BaseFolder, dlc.ExtendedInfo.FileName), Value: "base file is missing"})
			}
		}
	}
	for k, v := range localDB.Skipped {
		issues = append(issues, Pair{Key: filepath.Join(k.BaseFolder, k.FileName), Value: v.ReasonText})
	}

	response.LibraryData = libraryData
	response.NumFiles = localDB.NumFiles
	response.Issues = issues

	return response, nil
}

func (a *App) UpdateDB() error {
	a.state.Lock()
	defer a.state.Unlock()

	if a.state.switchDB == nil {
		switchDb, err := a.buildSwitchDb()
		if err != nil {
			a.sugarLogger.Error(err)
			return err
		}
		a.state.switchDB = switchDb
	}
	return nil
}

func (a *App) GetMissingUpdates() []process.IncompleteTitle {
	settingsObj := settings.ReadSettings(a.baseFolder)
	ignoreIds := map[string]struct{}{}
	for _, id := range settingsObj.IgnoreUpdateTitleIds {
		ignoreIds[strings.ToLower(id)] = struct{}{}
	}

	a.state.Lock()
	defer a.state.Unlock()

	if a.state.localDB == nil || a.state.switchDB == nil {
		return []process.IncompleteTitle{}
	}

	missingUpdates := process.ScanForMissingUpdates(a.state.localDB.TitlesMap, a.state.switchDB.TitlesMap, ignoreIds, settingsObj.IgnoreDLCUpdates)
	values := make([]process.IncompleteTitle, len(missingUpdates))
	i := 0
	for _, missingUpdate := range missingUpdates {
		values[i] = missingUpdate
		i++
	}
	return values
}

func (a *App) GetMissingDLC() []process.IncompleteTitle {
	settingsObj := settings.ReadSettings(a.baseFolder)
	ignoreIds := map[string]struct{}{}
	for _, id := range settingsObj.IgnoreDLCTitleIds {
		ignoreIds[strings.ToLower(id)] = struct{}{}
	}

	a.state.Lock()
	defer a.state.Unlock()

	if a.state.localDB == nil || a.state.switchDB == nil {
		return []process.IncompleteTitle{}
	}

	missingDLC := process.ScanForMissingDLC(a.state.localDB.TitlesMap, a.state.switchDB.TitlesMap, ignoreIds)
	values := make([]process.IncompleteTitle, len(missingDLC))
	i := 0
	for _, missingUpdate := range missingDLC {
		values[i] = missingUpdate
		i++
	}
	return values
}

func (a *App) CheckUpdate() bool {
	newUpdate, err := settings.CheckForUpdates()
	if err != nil {
		a.sugarLogger.Error(err)
		if !strings.Contains(err.Error(), "dial tcp") {
			runtime.EventsEmit(a.ctx, "error", err.Error())
		}
	}
	return newUpdate
}

// Helper methods

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

func (a *App) buildSwitchDb() (*db.SwitchTitlesDB, error) {
	settingsObj := settings.ReadSettings(a.baseFolder)
	//1. load the titles JSON object
	a.UpdateProgress(1, 4, "Downloading titles.json")
	filename := filepath.Join(a.baseFolder, settings.TITLE_JSON_FILENAME)
	titleFile, titlesEtag, err := db.LoadAndUpdateFile(settingsObj.TitlesJsonUrl, filename, settingsObj.TitlesEtag)
	if err != nil {
		return nil, errors.New("failed to download switch titles [reason:" + err.Error() + "]")
	}
	settingsObj.TitlesEtag = titlesEtag

	a.UpdateProgress(2, 4, "Downloading versions.json")
	filename = filepath.Join(a.baseFolder, settings.VERSIONS_JSON_FILENAME)
	versionsFile, versionsEtag, err := db.LoadAndUpdateFile(settingsObj.VersionsJsonUrl, filename, settingsObj.VersionsEtag)
	if err != nil {
		return nil, errors.New("failed to download switch updates [reason:" + err.Error() + "]")
	}
	settingsObj.VersionsEtag = versionsEtag

	settings.SaveSettings(settingsObj, a.baseFolder)

	a.UpdateProgress(3, 4, "Processing switch titles and updates ...")
	switchTitleDB, err := db.CreateSwitchTitleDB(titleFile, versionsFile)
	a.UpdateProgress(4, 4, "Finishing up...")
	return switchTitleDB, err
}

func (a *App) buildLocalDB(localDbManager *db.LocalSwitchDBManager, ignoreCache bool) (*db.LocalSwitchFilesDB, error) {
	folderToScan := settings.ReadSettings(a.baseFolder).Folder
	recursiveMode := settings.ReadSettings(a.baseFolder).ScanRecursively

	scanFolders := settings.ReadSettings(a.baseFolder).ScanFolders
	scanFolders = append(scanFolders, folderToScan)
	// Note: localDbManager.CreateLocalSwitchFilesDB expects a ProgressUpdater interface (g).
	// App needs to implement UpdateProgress.
	localDB, err := localDbManager.CreateLocalSwitchFilesDB(scanFolders, a, recursiveMode, ignoreCache)
	a.state.localDB = localDB
	return localDB, err
}

func (a *App) UpdateProgress(curr int, total int, message string) {
	progressMessage := ProgressUpdate{curr, total, message}
	a.sugarLogger.Debugf("%v (%v/%v)", message, curr, total)
	runtime.EventsEmit(a.ctx, "updateProgress", progressMessage)
}
