package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/trembon/switch-library-manager/console"
	"github.com/trembon/switch-library-manager/settings"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("failed to get executable directory, please ensure app has sufficient permissions. aborting")
		return
	}

	workingFolder := filepath.Dir(exePath)

	if runtime.GOOS == "darwin" {
		if strings.Contains(workingFolder, ".app") {
			appIndex := strings.Index(workingFolder, ".app")
			sepIndex := strings.LastIndex(workingFolder[:appIndex], string(os.PathSeparator))
			workingFolder = workingFolder[:sepIndex]
		}
	}

	appSettings := settings.ReadSettings(workingFolder)

	logger := createLogger(workingFolder, appSettings.Debug)

	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Info("[SLM starts]")
	sugar.Infof("[Executable: %v]", exePath)
	sugar.Infof("[Working directory: %v]", workingFolder)

	files, err := AssetDir(workingFolder)
	if files == nil && err == nil {
		appSettings.GUI = false
	}

	console.InitializeFlags()
	console.LogFlags(sugar)

	consoleFlags := console.GetFlagsValues()
	useGUI := appSettings.GUI
	if consoleFlags.Mode.IsSet() {
		mode := consoleFlags.Mode.String()
		if mode == "console" {
			useGUI = false
		} else if mode == "gui" {
			useGUI = true
		}
	}

	if useGUI {
		err := wails.Run(&options.App{
			Title:  "Switch Library Manager",
			Width:  1200,
			Height: 600,
			Windows: &windows.Options{
				Theme: windows.SystemDefault,
			},
			OnStartup: func(ctx *wails.Context) {
				gui := CreateGUI(workingFolder, sugar)
				gui.Start(ctx)
			},
		})
		if err != nil {
			sugar.Fatalf("Error starting GUI: %v", err)
		}
	} else {
		console.FixConsoleOutput()
		CreateConsole(workingFolder, sugar, consoleFlags).Start()
	}
}

func createLogger(workingFolder string, debug bool) *zap.Logger {
	var config zap.Config
	if debug {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logPath := filepath.Join(workingFolder, "slm.log")
	// delete old file
	os.Remove(logPath)

	if runtime.GOOS == "windows" {
		zap.RegisterSink("winfile", func(u *url.URL) (zap.Sink, error) {
			// Remove leading slash left by url.Parse()
			return zapcore.AddSync(&os.File{
				Name: u.Path[1:],
				Flag: os.O_WRONLY | os.O_APPEND | os.O_CREATE,
				Perm: 0644,
			}), nil
		})
		logPath = "winfile:///" + logPath
	}

	config.OutputPaths = []string{logPath}
	config.ErrorOutputPaths = []string{logPath}
	logger, err := config.Build()
	if err != nil {
		fmt.Printf("failed to create logger - %v", err)
		panic(1)
	}
	zap.ReplaceGlobals(logger)
	return logger
}
