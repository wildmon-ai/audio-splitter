# Audio Splitter

[![Build](https://img.shields.io/badge/build-electron--builder-blue)](#build)
[![Stack](https://img.shields.io/badge/stack-Vue%203%20%2B%20Electron%20%2B%20Go-42b883)](#architecture)
[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)](#build)

A cross-platform desktop application for batch-splitting long-form wildlife audio recordings into fixed-length clips, ready for acoustic monitoring and analysis pipelines.

Built with **Electron**, **Vue 3**, **TypeScript**, and a **Go** backend powered by `ffmpeg`/`ffprobe`.

## Features

- **Batch processing** of entire folder trees, preserving the original subfolder structure in the output.
- **Configurable sampling**: extract one fixed-length clip every N minutes from each recording.
- **Timestamp-aware naming**: when the source filename encodes a recording start time, output clips are named with the real-world clip start time.
- **Resumable**: files that have already been processed are skipped automatically.
- **Parallel processing** with configurable worker count, or automatic based on CPU cores.
- **Live progress and logs** streamed from the Go backend to the UI in real time.

## How It Works

Given a folder of audio files, the app extracts one 1-minute clip every N minutes, preserving the original subfolder structure in the output folder. Files already processed are skipped automatically.

Output filenames carry the real-world timestamp of the clip's start time when the source filename contains a timestamp. For example, `site_20240601_060000.wav` with `target_step=5` produces clips such as `site_20240601_060000.wav`, `site_20240601_060500.wav`, and `site_20240601_061000.wav`.

Files without a timestamp are still processed and named with the audio offset, such as `recording_0000m.wav`.

## Parameters

| Parameter | Default | Description |
| --- | --- | --- |
| **Extract Every N Minutes** (`target_step`) | 10 | Take one clip every N recorded minutes |
| **Record Step** (`record_step`) | 1 | Real-world minutes per audio minute, used when recorder time advances faster than the audio duration |
| **Parallel Processes** (`num_threads`) | 0 | Concurrent ffmpeg processes; `0` means auto, using all CPU cores |

Clip duration is always **1 minute**.

## Architecture

This is a three-layer desktop app: **Vue 3 frontend → Electron main process → Go backend**.

1. The user configures parameters in the Vue UI and clicks Start.
2. The renderer sends an IPC message (`start-processing`) via the context bridge (`electron/preload.ts`).
3. The Electron main process (`electron/main.ts`) forwards the request to the Go backend over stdin.
4. The Go backend (`backend/main.go`) uses `ffprobe` to detect audio duration, then spawns `ffmpeg` to split each file into clips. Progress and log lines are written to stdout as JSON.
5. Electron reads stdout line-by-line and forwards `progress`, `log`, and `done` events back to the renderer.
6. The Pinia store (`src/stores/processing.ts`) consumes these events and drives `ProgressBar.vue` and `LogPanel.vue`.

## Installation

Download the latest installer for your platform from the [Releases](../../releases) page:

| Platform | Artifact |
| --- | --- |
| Windows | `Audio Splitter Setup x.x.x.exe` |
| macOS | `Audio Splitter-x.x.x.dmg` |
| Linux | `Audio Splitter-x.x.x.AppImage` or `audio-splitter_x.x.x_amd64.deb` |

No additional dependencies are required — `ffmpeg`, `ffprobe`, and the Go backend are bundled inside the installer.

## Development

### Requirements

- [Node.js](https://nodejs.org) + [pnpm](https://pnpm.io)
- [Go](https://go.dev) 1.22+
- ffmpeg/ffprobe in `resources/`, or available on `PATH`

### Setup

```bash
pnpm install

# Download ffmpeg/ffprobe binaries for local dev (Linux)
bash scripts/download-ffmpeg.sh amd64
```

### Run

```bash
pnpm dev
```

`pnpm dev` builds the Go backend into `dist-backend/audio-splitter-backend`, starts Vite, compiles the Electron main process, and launches Electron. The backend looks for ffmpeg/ffprobe in `resources/` during development and falls back to `PATH`.

### Test

```bash
pnpm test
```

## Build

```bash
pnpm dist
```

This runs in order:

1. `go build` builds `backend/main.go` into `dist-backend/audio-splitter-backend`.
2. `vue-tsc`, `vite build`, and `tsc` compile the frontend and Electron main process.
3. `electron-builder` packages the desktop app into `release/`.

| Platform | Output |
| --- | --- |
| Linux | `release/Audio Splitter-x.x.x.AppImage`, `release/audio-splitter_x.x.x_amd64.deb` |
| macOS | `release/Audio Splitter-x.x.x.dmg` |
| Windows | `release/Audio Splitter Setup x.x.x.exe` |

The Go backend and ffmpeg binaries are bundled inside the installer when `resources/ffmpeg` and `resources/ffprobe` are present. End users do not need Go or ffmpeg installed separately.

## ffmpeg

Download static ffmpeg/ffprobe binaries and place them in `resources/`:

| OS | Source |
| --- | --- |
| Linux | [johnvansickle.com](https://johnvansickle.com/ffmpeg/) static build |
| macOS | [evermeet.cx](https://evermeet.cx/ffmpeg/) |
| Windows | [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds) |

On Linux, this helper downloads the static build into `resources/`:

```bash
bash scripts/download-ffmpeg.sh
```

The release workflow downloads ffmpeg/ffprobe automatically on each platform.

## Releasing

Releases are triggered by pushing a `v*` tag. The CI workflow (`.github/workflows/release.yml`):

1. Creates the GitHub release.
2. Builds in parallel on Linux, macOS, and Windows.
3. Sets the npm `version` field from the tag before building.
4. Downloads platform-specific ffmpeg/ffprobe static binaries.
5. Publishes artifacts to the GitHub release via `electron-builder --publish always`.

## Project Structure

```text
src/                        Vue 3 frontend
  components/
    FolderPicker.vue        Native folder selection
    ParameterForm.vue       Processing parameters
    ProgressBar.vue         Progress bar with current file
    LogPanel.vue            Scrollable log output
  stores/
    processing.ts           Pinia store and IPC event listeners
backend/
  main.go                   Go backend, ffprobe + ffmpeg via subprocess
electron/
  main.ts                   Electron main process, spawns Go backend
  preload.ts                Context bridge for IPC
resources/                  ffmpeg/ffprobe binaries, git-ignored
scripts/
  build-backend.cjs         Cross-platform Go backend build
  copy-backend.cjs          afterPack hook copying backend into app bundle
  download-ffmpeg.sh        Downloads ffmpeg/ffprobe for local dev (Linux)
  write-electron-package.cjs CommonJS marker for compiled Electron files
```
