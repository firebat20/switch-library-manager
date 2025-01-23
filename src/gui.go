package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/asticode/go-astilectron"
	"github.com/trembon/switch-library-manager/db"
	"github.com/trembon/switch-library-manager/process"
	"github.com/trembon/switch-library-manager/settings"
	"github.com/wailsapp/wails/v2"
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
	window   *astilectron.Window
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
	ctx            *wails.Context
}

func CreateGUI(baseFolder string, sugarLogger *zap.SugaredLogger) *GUI {
	return &GUI{state: State{}, baseFolder: baseFolder, sugarLogger: sugarLogger}
}

func (g *GUI) Start(ctx *wails.Context) {
	g.ctx = ctx

	localDbManager, err := db.NewLocalSwitchDBManager(g.baseFolder)
	if err != nil {
		g.sugarLogger.Error("Failed to create local files db\n", err)
		return
	}

	settings.InitSwitchKeys(g.baseFolder)

	g.localDbManager = localDbManager
	defer localDbManager.Close()

	runtime.EventsOn(g.ctx, "message", g.handleMessage)
}

func (g *GUI) handleMessage(data ...interface{}) {
	msg := data[0].(Message)
	var retValue string

	g.state.Lock()
	defer g.state.Unlock()

	g.sugarLogger.Debugf("Received message from client [%v]", msg)

	switch msg.Name {
	case "organize":
		g.organizeLibrary()
	case "isKeysFileAvailable":
		keys, _ := settings.SwitchKeys()
		retValue = strconv.FormatBool(keys != nil && keys.GetKey("header_key") != "")
	case "loadSettings":
		retValue = g.loadSettings()

		g.state.window.SetAlwaysOnTop(false)
	case "saveSettings":
		err := g.saveSettings(msg.Payload)
		if err != nil {
			g.sugarLogger.Error(err)
			runtime.EventsEmit(g.ctx, "error", err.Error())
			return
		}
	case "missingGames":
		missingGames := g.getMissingGames()
		msg, _ := json.Marshal(missingGames)
		runtime.EventsEmit(g.ctx, "missingGames", string(msg))
	case "updateLocalLibrary":
		ignoreCache, _ := strconv.ParseBool(msg.Payload)
		localDB, err := g.buildLocalDB(g.localDbManager, ignoreCache)
		if err != nil {
			g.sugarLogger.Error(err)
			runtime.EventsEmit(g.ctx, "error", err.Error())
			return
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

				if v.Updates != nil && len(v.Updates) != 0 {
					if v.Updates[v.LatestUpdate].Metadata.Ncap != nil {
						version = v.Updates[v.LatestUpdate].Metadata.Ncap.DisplayVersion
					} else {
						version = ""
					}
				}
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
		msg, _ := json.Marshal(response)
		runtime.EventsEmit(g.ctx, "libraryLoaded", string(msg))
	case "updateDB":
		if g.state.switchDB == nil {
			switchDb, err := g.buildSwitchDb()
			if err != nil {
				g.sugarLogger.Error(err)
				runtime.EventsEmit(g.ctx, "error", err.Error())
				return
			}
			g.state.switchDB = switchDb
		}
	case "missingUpdates":
		retValue = g.getMissingUpdates()
	case "missingDlc":
		retValue = g.getMissingDLC()
	case "checkUpdate":
		newUpdate, err := settings.CheckForUpdates()
		if err != nil {
			g.sugarLogger.Error(err)
			if !strings.Contains(err.Error(), "dial tcp") {
				runtime.EventsEmit(g.ctx, "error", err.Error())
			}
		}
		retValue = strconv.FormatBool(newUpdate)
	}

	g.sugarLogger.Debugf("Server response [%v]", retValue)
	runtime.EventsEmit(g.ctx, "response", retValue)
}

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

func (g *GUI) saveSettings(settingsJson string) error {
	s := settings.AppSettings{}
	err := json.Unmarshal([]byte(settingsJson), &s)
	if err != nil {
		return err
	}
	settings.SaveSettings(&s, g.baseFolder)
	return nil
}

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

func (g *GUI) loadSettings() string {
	return settings.ReadSettingsAsJSON(g.baseFolder)
}

func (g *GUI) buildSwitchDb() (*db.SwitchTitlesDB, error) {
	settingsObj := settings.ReadSettings(g.baseFolder)
	//1. load the titles JSON object
	g.UpdateProgress(1, 4, "Downloading titles.json")
	filename := filepath.Join(g.baseFolder, settings.TITLE_JSON_FILENAME)
	titleFile, titlesEtag, err := db.LoadAndUpdateFile(settingsObj.TitlesJsonUrl, filename, settingsObj.TitlesEtag)
	if err != nil {
		return nil, errors.New("failed to download switch titles [reason:" + err.Error() + "]")
	}
	settingsObj.TitlesEtag = titlesEtag

	g.UpdateProgress(2, 4, "Downloading versions.json")
	filename = filepath.Join(g.baseFolder, settings.VERSIONS_JSON_FILENAME)
	versionsFile, versionsEtag, err := db.LoadAndUpdateFile(settingsObj.VersionsJsonUrl, filename, settingsObj.VersionsEtag)
	if err != nil {
		return nil, errors.New("failed to download switch updates [reason:" + err.Error() + "]")
	}
	settingsObj.VersionsEtag = versionsEtag

	settings.SaveSettings(settingsObj, g.baseFolder)

	g.UpdateProgress(3, 4, "Processing switch titles and updates ...")
	switchTitleDB, err := db.CreateSwitchTitleDB(titleFile, versionsFile)
	g.UpdateProgress(4, 4, "Finishing up...")
	return switchTitleDB, err
}

func (g *GUI) buildLocalDB(localDbManager *db.LocalSwitchDBManager, ignoreCache bool) (*db.LocalSwitchFilesDB, error) {
	folderToScan := settings.ReadSettings(g.baseFolder).Folder
	recursiveMode := settings.ReadSettings(g.baseFolder).ScanRecursively

	scanFolders := settings.ReadSettings(g.baseFolder).ScanFolders
	scanFolders = append(scanFolders, folderToScan)
	localDB, err := localDbManager.CreateLocalSwitchFilesDB(scanFolders, g, recursiveMode, ignoreCache)
	g.state.localDB = localDB
	return localDB, err
}

func (g *GUI) organizeLibrary() {
	folderToScan := settings.ReadSettings(g.baseFolder).Folder
	options := settings.ReadSettings(g.baseFolder).OrganizeOptions
	if !process.IsOptionsValid(options) {
		zap.S().Error("the organize options in settings.json are not valid, please check that the template contains file/folder name")
		runtime.EventsEmit(g.ctx, "error", "the organize options in settings.json are not valid, please check that the template contains file/folder name")
		return
	}
	process.OrganizeByFolders(folderToScan, g.state.localDB, g.state.switchDB, g)
	if settings.ReadSettings(g.baseFolder).OrganizeOptions.DeleteOldUpdateFiles {
		process.DeleteOldUpdates(g.baseFolder, g.state.localDB, g)
	}
}

func (g *GUI) UpdateProgress(curr int, total int, message string) {
	progressMessage := ProgressUpdate{curr, total, message}
	g.sugarLogger.Debugf("%v (%v/%v)", message, curr, total)
	msg, err := json.Marshal(progressMessage)
	if err != nil {
		g.sugarLogger.Error(err)
		return
	}

	runtime.EventsEmit(g.ctx, "updateProgress", string(msg))
}

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
