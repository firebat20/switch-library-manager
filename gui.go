package main

import (
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
	"github.com/giwty/switch-library-manager/db"
	"github.com/giwty/switch-library-manager/process"
	"github.com/giwty/switch-library-manager/settings"
	"go.uber.org/zap"
	"log"
	"path/filepath"
	"strconv"
	"sync"
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

type LibraryTemplateData struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Version      int    `json:"version"`
	Dlc          string `json:"dlc"`
	TitleId      string `json:"titleId"`
	Path         string `json:"path"`
	Icon         string `json:"icon"`
	Update       int    `json:"update"`
	MultiContent bool   `json:"multi_content"`
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
}

func CreateGUI(baseFolder string, sugarLogger *zap.SugaredLogger) *GUI {
	return &GUI{state: State{}, baseFolder: baseFolder, sugarLogger: sugarLogger}
}
func (g *GUI) Start() {

	localDbManager, err := db.NewLocalSwitchDBManager(g.baseFolder)
	if err != nil {
		g.sugarLogger.Error("Failed to create local files db\n", err)
		return
	}

	settings.InitSwitchKeys(g.baseFolder)

	g.localDbManager = localDbManager
	defer localDbManager.Close()
	// Run bootstrap
	if err := bootstrap.Run(bootstrap.Options{
		Asset:    Asset,
		AssetDir: AssetDir,
		AstilectronOptions: astilectron.Options{
			AppName:            "Switch Library Manager",
			AppIconDarwinPath:  "resources/icon.icns",
			AppIconDefaultPath: "resources/icon.png",
			SingleInstance:     true,
			//VersionAstilectron: VersionAstilectron,
			//VersionElectron:    VersionElectron,
		},
		Debug:         false,
		Logger:        log.New(log.Writer(), log.Prefix(), log.Flags()),
		RestoreAssets: RestoreAssets,
		Windows: []*bootstrap.Window{{
			Homepage: "app.html",
			Adapter: func(w *astilectron.Window) {
				g.state.window = w
				g.state.window.OnMessage(g.handleMessage)
				//g.state.window.OpenDevTools()
			},
			Options: &astilectron.WindowOptions{
				BackgroundColor: astikit.StrPtr("#333"),
				Center:          astikit.BoolPtr(true),
				Height:          astikit.IntPtr(600),
				Width:           astikit.IntPtr(1200),
			},
		}},
		MenuOptions: []*astilectron.MenuItemOptions{{
			Label: astikit.StrPtr("Debug"),
			SubMenu: []*astilectron.MenuItemOptions{
				{
					Label: astikit.StrPtr("Open DevTools"),
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						g.state.window.OpenDevTools()
						return
					},
				},
				{Role: astilectron.MenuItemRoleClose},
			},
		}},
	}); err != nil {
		g.sugarLogger.Error(fmt.Errorf("running bootstrap failed: %w", err))
	}
}

func (g *GUI) handleMessage(m *astilectron.EventMessage) interface{} {
	var retValue string
	g.state.Lock()
	defer g.state.Unlock()
	msg := Message{}
	err := m.Unmarshal(&msg)

	if err != nil {
		g.sugarLogger.Error("Failed to parse client message", err)
		return ""
	}

	g.sugarLogger.Debugf("Received message from client [%v]", msg)

	switch msg.Name {
	case "organize":
		g.organizeLibrary()
	case "isKeysFileAvailable":
		keys, _ := settings.SwitchKeys()
		retValue = strconv.FormatBool(keys != nil && keys.GetKey("header_key") != "")
	case "loadSettings":
		retValue = g.loadSettings()
	case "saveSettings":
		err = g.saveSettings(msg.Payload)
		if err != nil {
			g.sugarLogger.Error(err)
			g.state.window.SendMessage(Message{Name: "error", Payload: err.Error()}, func(m *astilectron.EventMessage) {})
			return ""
		}
	case "updateLocalLibrary":
		localDB, err := g.buildLocalDB(g.localDbManager)
		if err != nil {
			g.sugarLogger.Error(err)
			g.state.window.SendMessage(Message{Name: "error", Payload: err.Error()}, func(m *astilectron.EventMessage) {})
			return ""
		}
		response := LocalLibraryData{}
		libraryData := []LibraryTemplateData{}
		issues := []Pair{}
		for k, v := range localDB.TitlesMap {
			if v.BaseExist {
				if title, ok := g.state.switchDB.TitlesMap[k]; ok {
					libraryData = append(libraryData,
						LibraryTemplateData{
							Icon:         title.Attributes.IconUrl,
							Name:         title.Attributes.Name,
							TitleId:      v.File.Metadata.TitleId,
							Update:       v.LatestUpdate,
							MultiContent: v.MultiContent,
							Path:         filepath.Join(v.File.ExtendedInfo.BaseFolder, v.File.ExtendedInfo.Info.Name()),
						})
				} else {
					libraryData = append(libraryData,
						LibraryTemplateData{
							Name:    db.ParseTitleNameFromFileName(v.File.ExtendedInfo.Info.Name()),
							TitleId: v.File.Metadata.TitleId,
							Path:    v.File.ExtendedInfo.Info.Name(),
						})
				}

			} else {
				for _, update := range v.Updates {
					issues = append(issues, Pair{Key: filepath.Join(update.ExtendedInfo.BaseFolder, update.ExtendedInfo.Info.Name()), Value: "base file is missing"})
				}
				for _, dlc := range v.Dlc {
					issues = append(issues, Pair{Key: filepath.Join(dlc.ExtendedInfo.BaseFolder, dlc.ExtendedInfo.Info.Name()), Value: "base file is missing"})
				}
			}
		}
		for k, v := range localDB.Skipped {
			issues = append(issues, Pair{Key: filepath.Join(k.BaseFolder, k.Info.Name()), Value: v})
		}

		response.LibraryData = libraryData
		response.NumFiles = localDB.NumFiles
		response.Issues = issues
		msg, _ := json.Marshal(response)
		g.state.window.SendMessage(Message{Name: "libraryLoaded", Payload: string(msg)}, func(m *astilectron.EventMessage) {})
	case "updateDB":
		if g.state.switchDB == nil {
			switchDb, err := g.buildSwitchDb()
			if err != nil {
				g.sugarLogger.Error(err)
				g.state.window.SendMessage(Message{Name: "error", Payload: err.Error()}, func(m *astilectron.EventMessage) {})
				return ""
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
			g.state.window.SendMessage(Message{Name: "error", Payload: err.Error()}, func(m *astilectron.EventMessage) {})
			return ""
		}
		retValue = strconv.FormatBool(newUpdate)
	}

	g.sugarLogger.Debugf("Server response [%v]", retValue)

	return retValue
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
	missingDLC := process.ScanForMissingDLC(g.state.localDB.TitlesMap, g.state.switchDB.TitlesMap)
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
	missingUpdates := process.ScanForMissingUpdates(g.state.localDB.TitlesMap, g.state.switchDB.TitlesMap)
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
	titleFile, titlesEtag, err := db.LoadAndUpdateFile(settings.TITLES_JSON_URL, filename, settingsObj.TitlesEtag)
	if err != nil {
		return nil, err
	}
	settingsObj.TitlesEtag = titlesEtag

	g.UpdateProgress(2, 4, "Downloading versions.json")
	filename = filepath.Join(g.baseFolder, settings.VERSIONS_JSON_FILENAME)
	versionsFile, versionsEtag, err := db.LoadAndUpdateFile(settings.VERSIONS_JSON_URL, filename, settingsObj.VersionsEtag)
	if err != nil {
		return nil, err
	}
	settingsObj.VersionsEtag = versionsEtag

	settings.SaveSettings(settingsObj, g.baseFolder)

	g.UpdateProgress(3, 4, "Processing switch titles and updates ...")
	switchTitleDB, err := db.CreateSwitchTitleDB(titleFile, versionsFile)
	g.UpdateProgress(4, 4, "Finishing up...")
	return switchTitleDB, err
}

func (g *GUI) buildLocalDB(localDbManager *db.LocalSwitchDBManager) (*db.LocalSwitchFilesDB, error) {
	folderToScan := settings.ReadSettings(g.baseFolder).Folder
	recursiveMode := settings.ReadSettings(g.baseFolder).ScanRecursively

	scanFolders := settings.ReadSettings(g.baseFolder).ScanFolders
	scanFolders = append(scanFolders, folderToScan)
	localDB, err := localDbManager.CreateLocalSwitchFilesDB(scanFolders, g, recursiveMode)
	g.state.localDB = localDB
	return localDB, err
}

func (g *GUI) organizeLibrary() {
	folderToScan := settings.ReadSettings(g.baseFolder).Folder
	process.OrganizeByFolders(folderToScan, g.state.localDB, g.state.switchDB, g)
	if settings.ReadSettings(g.baseFolder).OrganizeOptions.DeleteOldUpdateFiles {
		process.DeleteOldUpdates(g.state.localDB, g)
	}
}

func (g *GUI) UpdateProgress(curr int, total int, message string) {
	progressMessage := ProgressUpdate{curr, total, message}
	g.sugarLogger.Debugf("process %v (%v/%v)", message, curr, total)
	msg, err := json.Marshal(progressMessage)
	if err != nil {
		g.sugarLogger.Error(err)
		return
	}

	g.state.window.SendMessage(Message{Name: "updateProgress", Payload: string(msg)}, func(m *astilectron.EventMessage) {})
}
