## Windows SmartScreen Warning

When running for the first time, Windows may show a SmartScreen warning. 
This is because the app is not yet code signed. Click **More info** → **Run anyway** to proceed.

# Auto BallChasing

A lightweight Windows system tray app that automatically uploads your Rocket League replay files to [ballchasing.com](https://ballchasing.com) as soon as they are saved.

## Lightweight for placebo merchants

<img width="724" height="132" alt="image" src="https://github.com/user-attachments/assets/9d62b0a8-7680-485c-bb73-e0199e3ba42f" />

## Features

- Watches your Rocket League replay folder automatically
- Uploads **saved** replays to ballchasing.com instantly (Auto save isn't a thing!)
- Titles each replay with an ISO timestamp
- Handles duplicate replays gracefully
- Saves your API key and visibility settings between sessions
- Optional start on Windows login

## Download

Go to [Releases](https://github.com/liamdoocey/autoballchasing/releases) and download the latest `AutoBallChasing.exe`.

## Setup

1. Download `AutoBallChasing.exe` from the releases page
2. Run it, an icon will appear in your system tray
3. Get your API key from [ballchasing.com/upload](https://ballchasing.com/upload)
4. Enter your API key in the settings window
5. Choose your preferred replay visibility (public / unlisted / private)
6. Click **Save Settings**
7. Click **Start Watcher**

From this point on, every replay you save in Rocket League will be uploaded automatically.

## Replay Folder

The app watches the default Rocket League replay folder: `Documents\My Games\Rocket League\TAGame\Demos` (NOTHING ELSE!)

## Visibility Options

| Option | Description |
|---|---|
| Public | Anyone can find and view your replays |
| Unlisted | Only accessible via direct link |
| Private | Only you can see them |

## Start on Login

Tick the **Start on login** checkbox to have the app launch automatically with Windows. Your replays will be uploaded even if you forget to open the app manually.

## Ballchasing API limits

<img width="1377" height="381" alt="image" src="https://github.com/user-attachments/assets/f56f3e2e-ae77-4127-9fab-39f1459bbbc5" />


## Building from Source

You will need:
- [Go 1.21+](https://go.dev/dl/)
- [rsrc](https://github.com/akavel/rsrc) for embedding the manifest

```bash
# Clone the repo
git clone https://github.com/yourusername/auto_ballchasing.git
cd auto_ballchasing

# Install dependencies
go mod tidy

# Generate the resource file
rsrc -manifest auto_ballchasing.manifest -ico icon.ico -o rsrc.syso

# Build
go build -ldflags="-H windowsgui -s -w" -o AutoBallChasing.exe .
```

## Running Tests

```bash
go test ./...
```

## Tech Stack

- [Go](https://go.dev/) - core language
- [fsnotify](https://github.com/fsnotify/fsnotify) - file system watching
- [walk](https://github.com/lxn/walk) - native Windows UI and system tray
- [ballchasing.com API](https://ballchasing.com/doc/api) - replay upload

## Privacy

Your API key is stored locally in: `%APPDATA%\auto_ballchasing\config.json`

It is never transmitted anywhere other than directly to ballchasing.com.
