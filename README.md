# Audio Splitter

[![Build](https://img.shields.io/badge/build-electron--builder-blue)](#build)
[![Stack](https://img.shields.io/badge/stack-Vue%203%20%2B%20Electron%20%2B%20Go-42b883)](#architecture)
[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)](#build)

A cross-platform desktop application for batch-splitting long-form wildlife audio recordings into fixed-length clips, ready for acoustic monitoring and analysis pipelines.

Built with **Electron**, **Vue 3**, **TypeScript**, and a **Go** backend powered by `ffmpeg`/`ffprobe`.

## Features

- **Batch processing** of entire folder trees, preserving the original subfolder structure in the output.
- **Configurable sampling**: extract one 1-minute clip every N minutes from each recording.
- **Timestamp-aware naming**: when the source filename encodes a recording start time, output clips are named with the real-world clip start time.
- **Resumable**: clips that already exist are skipped, and incomplete ones are made again.
- **Safe to stop**: clips are written to a temporary file and renamed when complete, so a cancel or crash never leaves a half-written clip.
- **Parallel processing** with configurable worker count, or automatic based on CPU cores.
- **Live progress and logs** streamed from the Go backend to the UI.
- **Help** menu (F1) with a usage guide inside the app.

## How It Works

Given a folder of audio files (WAV, FLAC, MP3, OGG, M4A, OPUS), the app extracts one 1-minute clip every N minutes, preserving the original subfolder structure in the output folder.

A clip is only made when at least 59 seconds of audio are left. Recorders often write files a few milliseconds longer than nominal (a "5-minute" file of 300.032 s): those do not get an extra, nearly empty clip, whose timestamped name would collide with the next recording's first clip.

Output filenames carry the real-world timestamp of the clip's start time when the source filename contains a `YYYYMMDD_HHMMSS` timestamp. For example, `site_20240601_060000.wav` with `target_step=5` produces clips such as `site_20240601_060000.wav`, `site_20240601_060500.wav`, and `site_20240601_061000.wav`. If two overlapping recordings would produce the same clip name, only the first is kept and the log says so.

Files without a timestamp are named with the audio offset in minutes, such as `recording_0000m.wav`.

Clips are mono, keep the source's file type, and keep the bit depth of WAV and FLAC recordings.

The scan skips hidden files (including the `._*.wav` files macOS writes on memory cards), `$RECYCLE.BIN` and `System Volume Information`, and the output folder when it is inside the source folder. Folders that cannot be read are skipped with a warning.

## Parameters

| Parameter | Default | Description |
| --- | --- | --- |
| **Extract Every N Minutes** (`target_step`) | 10 | Minutes from the start of one clip to the start of the next (1–100,000) |
| **Record Step** (`record_step`) | 1 | Real-world minutes per audio minute, for recorders whose clock advances faster than the audio. Not shown in the UI |
| **Parallel Processes** (`num_threads`) | 0 | Concurrent ffmpeg processes; `0` means one per CPU core, and values above 4× the core count are capped |

Clip duration is always **1 minute**. Invalid values are rejected by both the UI and the backend.

## Architecture

This is a three-layer desktop app: **Vue 3 frontend → Electron main process → Go backend**.

1. The user configures parameters in the Vue UI and clicks Start.
2. The renderer sends an IPC message (`start-processing`) via the context bridge (`electron/preload.ts`).
3. The Electron main process (`electron/main.ts`) starts the Go backend on demand and forwards the request over stdin.
4. The Go backend (`backend/`) uses `ffprobe` to read each file's duration, plans the clips, and runs `ffmpeg` in a worker pool. Progress (throttled) and log lines are written to stdout as JSON.
5. Electron reads stdout line by line and forwards `progress`, `log`, and `done` events to the renderer. If the backend exits mid-job, Electron reports it to the renderer, and the next job starts a new backend.
6. The Pinia store (`src/stores/processing.ts`) consumes these events and drives `ProgressBar.vue` and `LogPanel.vue`.

Every `start` command ends with exactly one `done` event, which carries `cancelled` or `error` when the job did not complete. Closing the backend's stdin cancels the job, stops ffmpeg, and exits.

## Installation

Download the latest installer for your platform from the [Releases](../../releases) page:

| Platform | Artifact |
| --- | --- |
| Windows | `Audio Splitter Setup x.x.x.exe` |
| macOS | `Audio Splitter-x.x.x.dmg` (Intel) or `Audio Splitter-x.x.x-arm64.dmg` (Apple Silicon) |
| Linux | `Audio Splitter-x.x.x.AppImage` or `audio-splitter_x.x.x_amd64.deb` (also arm64 and armv7l) |

No additional dependencies are required: `ffmpeg`, `ffprobe`, and the Go backend are bundled inside the installer.

## Development

### Requirements

- [Node.js](https://nodejs.org) 24 + [pnpm](https://pnpm.io) (the version in `packageManager`, via `corepack enable`)
- [Go](https://go.dev) 1.26+
- ffmpeg/ffprobe in `resources/<arch>/`, or available on `PATH`

### Setup

```bash
pnpm install

# ffmpeg/ffprobe for local development, into resources/<arch>/
bash scripts/download-ffmpeg.sh x64           # Linux (x64, arm64 or armv7l)
bash scripts/download-ffmpeg-macos.sh arm64   # macOS (arm64 or x64)
pwsh scripts/download-ffmpeg-windows.ps1      # Windows (x64)
```

### Run

```bash
pnpm dev
```

`pnpm dev` builds the Go backend for this computer into `dist-backend/audio-splitter-backend`, starts Vite, compiles the Electron main process, and launches Electron. The backend looks for ffmpeg/ffprobe in `resources/<arch>/` (then `resources/`) and falls back to `PATH`.

### Test

```bash
pnpm typecheck   # vue-tsc and the Electron main process
pnpm test        # frontend unit tests (Vitest)
pnpm test:go     # backend tests; those that run ffmpeg are skipped if it is not on PATH
```

CI (`.github/workflows/ci.yml`) runs all of them on Linux, macOS, and Windows for every push to `main` and every pull request.

## Build

```bash
pnpm dist
```

This runs in order:

1. `pnpm build:backend:release` builds the backend for every CPU this OS packages into `dist-backend/audio-splitter-backend-<arch>`.
2. `vue-tsc`, `vite build`, and `tsc` compile the frontend and Electron main process.
3. `electron-builder` packages the app into `release/`. Its `afterPack` hook (`scripts/copy-backend.cjs`) copies the matching backend into the app and **fails the build if ffmpeg/ffprobe for that CPU are missing** from `resources/<arch>/`.

| Platform | CPUs | Output |
| --- | --- | --- |
| Linux | x64, arm64, armv7l | `release/Audio Splitter-x.x.x.AppImage`, `release/audio-splitter_x.x.x_amd64.deb` |
| macOS | x64, arm64 | `release/Audio Splitter-x.x.x.dmg` |
| Windows | x64 | `release/Audio Splitter Setup x.x.x.exe` |

## ffmpeg

The download scripts put static builds in `resources/<arch>/` (electron-builder's names: `x64`, `arm64`, `armv7l`) and check them against published checksums:

| OS | Source | Script |
| --- | --- | --- |
| Linux | [johnvansickle.com](https://johnvansickle.com/ffmpeg/) static build (latest release, MD5 checked) | `scripts/download-ffmpeg.sh` |
| macOS | [ffmpeg.martin-riedl.de](https://ffmpeg.martin-riedl.de) static build (pinned version and SHA-256) | `scripts/download-ffmpeg-macos.sh` |
| Windows | [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds) release branch (SHA-256 checked) | `scripts/download-ffmpeg-windows.ps1` |

Do not bundle Homebrew's ffmpeg on macOS: it depends on Homebrew's libraries and fails on Macs without them. The release workflow rejects any macOS binary that links to libraries outside the system.

## Releasing

Releases are triggered by pushing a `v*` tag. The release workflow (`.github/workflows/release.yml`):

1. Runs the CI tests.
2. Creates a **draft** GitHub release.
3. Builds in parallel on Linux, macOS, and Windows, for every CPU listed above, after setting the npm `version` from the tag and downloading ffmpeg for each CPU.
4. Uploads the installers to the draft via `electron-builder --publish always`. Only the macOS job receives the Apple signing and notarization secrets.
5. Publishes the release once every platform has succeeded, so a failed build never shows up as the latest release.

## Project Structure

```text
src/                        Vue 3 frontend
  components/
    FolderPicker.vue        Native folder selection
    ParameterForm.vue       Processing parameters, with validation
    ProgressBar.vue         Scan and clip progress
    LogPanel.vue            Scrollable log output
  composables/
    usePersisted.ts         Settings kept in localStorage
  stores/
    processing.ts           Pinia store and IPC event listeners
  utils/                    Validation limits and formatting
public/
  help.html                 Help window content
backend/                    Go backend (ffprobe + ffmpeg via subprocess)
  main.go                   stdin/stdout protocol and job control
  config.go                 Job settings and folder checks
  scan.go                   Finding the recordings
  plan.go                   Which clips to cut and their names
  media.go                  ffmpeg/ffprobe
  run.go                    Running a job: plan, worker pool, progress
electron/
  main.ts                   Electron main process, backend lifecycle, menus
  preload.ts                Context bridge for IPC
resources/                  ffmpeg/ffprobe per CPU (resources/<arch>/), git-ignored
scripts/
  build-backend.cjs         Go backend build (this computer, or --release for every CPU)
  copy-backend.cjs          afterPack hook: bundles the backend, checks ffmpeg is bundled
  download-ffmpeg.sh        ffmpeg for Linux
  download-ffmpeg-macos.sh  ffmpeg for macOS
  download-ffmpeg-windows.ps1  ffmpeg for Windows
  write-electron-package.cjs   CommonJS marker for compiled Electron files
```
