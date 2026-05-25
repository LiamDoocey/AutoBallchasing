package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"auto_ballchasing/config"
	"auto_ballchasing/uploader"
	"auto_ballchasing/watcher"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"golang.org/x/sys/windows/registry"
)

var (
	mainWindow    *walk.MainWindow
	apiKeyEdit    *walk.LineEdit
	visibilityBox *walk.ComboBox
	statusLabel   *walk.Label
	okLabel       *walk.Label
	dupLabel      *walk.Label
	errLabel      *walk.Label
	logBox        *walk.ListBox
	toggleBtn     *walk.PushButton
	startupCheck  *walk.CheckBox
)

var (
	cfg            *config.Config
	currentWatcher *watcher.Watcher
	running        bool
	okCount        int
	dupCount       int
	errCount       int
	logEntries     []string
)

// activityLog implements walk.ListModel for the activity log
type activityLog struct {
	walk.ListModelBase
}

func (m *activityLog) ItemCount() int          { return len(logEntries) }
func (m *activityLog) Value(i int) interface{} { return logEntries[i] }

var logModel = new(activityLog)

func main() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Printf("could not load config: %v", err)
		cfg = &config.Config{Visibility: "public"}
	}

	if err := runUI(); err != nil {
		log.Fatal(err)
	}
}

func runUI() error {
	visibilityOptions := []string{"public", "unlisted", "private"}

	err := MainWindow{
		AssignTo: &mainWindow,
		Title:    "Auto BallChasing",
		MinSize:  Size{Width: 400, Height: 550},
		Layout:   VBox{},
		Children: []Widget{

			//-----Config section-----
			GroupBox{
				Title:  "Configuration",
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "API Key:"},
					LineEdit{
						AssignTo:     &apiKeyEdit,
						Text:         cfg.APIKey,
						PasswordMode: true,
					},

					Label{Text: "Visibility:"},
					ComboBox{
						AssignTo: &visibilityBox,
						Model:    visibilityOptions,
						Value:    cfg.Visibility,
					},

					Label{Text: "Run on startup?:"},
					CheckBox{
						AssignTo: &startupCheck,
						Checked:  getStartup(),
						OnCheckedChanged: func() {
							if err := setStartup(startupCheck.Checked()); err != nil {
								walk.MsgBox(mainWindow, "Error",
									"Could not update startup setting: "+err.Error(),
									walk.MsgBoxIconError)
							}
						},
					},

					PushButton{
						Text:      "Save Settings",
						OnClicked: saveSettings,
					},
					PushButton{
						AssignTo: &toggleBtn,
						Text:     "Start Watcher",
						OnClicked: func() {
							if running {
								stopWatcher()
							} else {
								startWatcher()
							}
						},
					},
				},
			},

			//-----Status section-----
			GroupBox{
				Title:  "Status",
				Layout: VBox{},
				Children: []Widget{
					Label{AssignTo: &statusLabel, Text: "Idle"},
					Label{AssignTo: &okLabel, Text: "Uploaded:   0"},
					Label{AssignTo: &dupLabel, Text: "Duplicates: 0"},
					Label{AssignTo: &errLabel, Text: "Failed:     0"},
				},
			},

			//-----Activity log-----
			GroupBox{
				Title:  "Activity",
				Layout: VBox{},
				Children: []Widget{
					ListBox{
						AssignTo: &logBox,
						Model:    logModel,
					},
				},
			},
		},
	}.Create()

	if err != nil {
		return fmt.Errorf("could not create window: %w", err)
	}

	// System tray
	trayIcon, err := walk.NewNotifyIcon(mainWindow)
	if err != nil {
		return fmt.Errorf("could not create tray icon: %w", err)
	}
	defer trayIcon.Dispose()

	trayIcon.SetToolTip("Auto BallChasing")
	trayIcon.SetVisible(true)

	// Tray left-click opens window
	trayIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mainWindow.Show()
			mainWindow.Activate()
		}
	})

	// Tray right-click menu
	openAction := walk.NewAction()
	openAction.SetText("Open")
	openAction.Triggered().Attach(func() {
		mainWindow.Show()
		mainWindow.Activate()
	})

	quitAction := walk.NewAction()
	quitAction.SetText("Quit")
	quitAction.Triggered().Attach(func() {
		stopWatcher()
		walk.App().Exit(0)
	})

	// Add actions directly to the tray icon
	trayIcon.ContextMenu().Actions().Add(openAction)
	trayIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	trayIcon.ContextMenu().Actions().Add(quitAction)

	// Hide to tray on close
	mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mainWindow.Hide()
	})

	// Auto-start if config is valid
	if cfg.IsValid() {
		startWatcher()
	}

	if icon := loadIcon(); icon != nil {
		trayIcon.SetIcon(icon)
		mainWindow.SetIcon(icon)
	}

	mainWindow.Run()
	return nil
}

func saveSettings() {
	cfg.APIKey = apiKeyEdit.Text()
	cfg.Visibility = visibilityBox.Text()
	if err := cfg.Save(); err != nil {
		walk.MsgBox(mainWindow, "Error", "Could not save settings: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(mainWindow, "Saved", "Settings saved successfully.", walk.MsgBoxIconInformation)
}

func startWatcher() {
	if running {
		return
	}

	key := apiKeyEdit.Text()
	if key == "" {
		walk.MsgBox(mainWindow, "Error", "API key is required.", walk.MsgBoxIconError)
		return
	}

	vis := visibilityBox.Text()
	u := uploader.New(key, uploader.Visibility(vis))

	if err := u.Ping(); err != nil {
		walk.MsgBox(mainWindow, "Error", "Could not connect to ballchasing: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	cfg.APIKey = key
	cfg.Visibility = vis
	if err := cfg.Save(); err != nil {
		log.Printf("could not save config: %v", err)
	}

	wt, err := watcher.New(replayFolder(), u)
	if err != nil {
		walk.MsgBox(mainWindow, "Error", "Could not watch folder: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	wt.Start()
	currentWatcher = wt
	running = true
	statusLabel.SetText("Watching...")
	toggleBtn.SetText("Stop Watcher")

	go func() {
		for result := range wt.Results {
			mainWindow.Synchronize(func() {
				if result.Duplicate {
					dupCount++
					dupLabel.SetText(fmt.Sprintf("Duplicates: %d", dupCount))
					addLog(fmt.Sprintf("~ %s (duplicate)", result.Filename))
				} else if result.Success {
					okCount++
					okLabel.SetText(fmt.Sprintf("Uploaded:   %d", okCount))
					addLog(fmt.Sprintf("✓ %s", result.Filename))
				} else {
					errCount++
					errLabel.SetText(fmt.Sprintf("Failed:     %d", errCount))
					addLog(fmt.Sprintf("✗ %s — %s", result.Filename, result.Error))
				}
			})
		}
	}()
}

func stopWatcher() {
	if currentWatcher != nil {
		currentWatcher.Stop()
		currentWatcher = nil
	}
	running = false
	statusLabel.SetText("Idle")
	toggleBtn.SetText("Start Watcher")
}

func addLog(entry string) {
	logEntries = append(logEntries, entry)
	logModel.PublishItemsReset()
	logBox.SetCurrentIndex(len(logEntries) - 1)
}

func replayFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("could not get home dir: %v", err)
		return ""
	}
	return filepath.Join(home, "Documents", "My Games", "Rocket League", "TAGame", "Demos")
}

func loadIcon() *walk.Icon {
	icon, err := walk.NewIconFromResourceId(2)
	if err != nil {
		log.Printf("could not load icon: %v", err)
		return nil
	}
	return icon
}

func setStartup(enable bool) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		// Get the path of the current executable
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return key.SetStringValue("AutoBallChasing", exe)
	}

	return key.DeleteValue("AutoBallChasing")
}

func getStartup() bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue("AutoBallChasing")
	return err == nil
}
