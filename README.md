# Audio Splitter

[![Build](https://img.shields.io/badge/build-electron--builder-blue)](#build)
[![Stack](https://img.shields.io/badge/stack-Vue%203%20%2B%20Electron%20%2B%20Go-42b883)](#architecture)
[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)](#build)

Audio Splitter takes short samples from long wildlife recordings. From every recording in a folder, it saves a 1-minute clip every few minutes, ready to listen to or to analyze. It runs on Windows, macOS and Linux, and never changes your recordings.

- **Using the app:** [Download](#download) · [How to use](#how-to-use) · [Choosing “Extract Every N Minutes”](#choosing-extract-every-n-minutes) · [Recordings saved in several files](#recordings-saved-in-several-files) · [How clips are named](#how-clips-are-named) · [Questions](#questions)
- **Changing the app:** [Architecture](#architecture) · [Development](#development) · [Build](#build) · [Releasing](#releasing) · [Troubleshooting](#troubleshooting)

## Download

Download the file for your computer from the [Releases](../../releases) page. In the names below, `x.y.z` is the version number.

| Your computer | File |
| --- | --- |
| Windows | `Audio-Splitter-Setup-x.y.z.exe` |
| Mac with an Apple chip (M1, M2, M3 …) | `Audio-Splitter-x.y.z-arm64.dmg` |
| Mac with an Intel processor | `Audio-Splitter-x.y.z.dmg` |
| Linux PC | `Audio-Splitter-x.y.z.AppImage`, or `audio-splitter_x.y.z_amd64.deb` for Ubuntu and Debian |
| Linux on ARM, 64-bit (for example Raspberry Pi OS 64-bit) | `Audio-Splitter-x.y.z-arm64.AppImage` or `audio-splitter_x.y.z_arm64.deb` |
| Linux on ARM, 32-bit (for example Raspberry Pi OS 32-bit) | `Audio-Splitter-x.y.z-armv7l.AppImage` or `audio-splitter_x.y.z_armv7l.deb` |

Not sure which Mac you have? Open the Apple menu → **About This Mac**: “Chip Apple M…” means an Apple chip, and “Processor … Intel” means Intel.

Everything the app needs is included, so there is nothing else to install. In the app, press **F1** (or choose **Help → How to Use Audio Splitter**) for a guide like this one.

## How to use

1. Next to **Source Folder**, click **Browse** and choose the folder with your recordings, for example the recorder's memory card. Recordings in its subfolders are included.
2. Next to **Output Folder**, click **Browse** and choose a folder for the clips. An empty folder is best. It cannot be the Source Folder.
3. Set **Extract Every N Minutes**: the time from the start of one clip to the start of the next (see below).
4. Leave **Parallel Processes** at `0`.
5. Click **Start Processing**. The progress bar shows how far it is, and the Log shows what is happening.
6. When it finishes, a summary tells you how many clips were saved, for example: **Done.** 12 clips extracted, 0 skipped, 0 errors · 41s

Your recordings are never changed or deleted. The clips are new files, saved in the Output Folder.

## Choosing “Extract Every N Minutes”

N is the time from the **start** of one clip to the **start** of the next. Every clip is 1 minute long, so after each clip the next N − 1 minutes are skipped.

> **For one clip every 5 minutes, enter 5, not 4.** For one clip every 10 minutes, enter 10.

With N = 5, a recording gives:

| Clip | Starts at | Ends at | Then skipped |
| --- | --- | --- | --- |
| 1st | minute 0 | minute 1 | minutes 1 to 5 |
| 2nd | minute 5 | minute 6 | minutes 6 to 10 |
| 3rd | minute 10 | minute 11 | minutes 11 to 15 |
| … | | | |

What you get from a 1-hour recording that starts at 14:00:

| N | Clips | Clips start at |
| --- | --- | --- |
| 1 | 60 | every minute: 14:00, 14:01, 14:02 … 14:59 (the whole hour) |
| 5 | 12 | 14:00, 14:05, 14:10 … 14:55 |
| 10 | 6 | 14:00, 14:10, 14:20, 14:30, 14:40, 14:50 |
| 15 | 4 | 14:00, 14:15, 14:30, 14:45 |
| 30 | 2 | 14:00, 14:30 |
| 60 | 1 | 14:00 |

A clip is only made when a full minute of recording is left. A recording of 10 minutes and 30 seconds with N = 5 gives clips at minutes 0 and 5: at minute 10 only 30 seconds are left, so there is no third clip. Recordings shorter than a minute give no clips.

## Recordings saved in several files

Many recorders save a new file every few minutes, for example every 5 minutes. Audio Splitter treats each file as a separate recording: the count starts again at the start of every file, so each file's first clip is its first minute.

Recordings saved as 5-minute files, starting at 14:00, 14:05, 14:10 …:

| N | Clips from each file | Clips start at | Spacing |
| --- | --- | --- | --- |
| 5 | 1, the first minute | 14:00, 14:05, 14:10 … | even |
| 1 | 5, every minute | 14:00, 14:01 … 14:04, 14:05 … | even |
| 4 | 2, at minutes 0 and 4 | 14:00, 14:04, 14:05, 14:09, 14:10 … | **uneven** |
| 10 | 1, the first minute | 14:00, 14:05, 14:10 … | every 5 minutes, not 10 |

> **For evenly spaced clips, choose an N that fits exactly into the length of your files.** For 5-minute files, use 1 or 5. For 1-hour files, use 1, 2, 3, 4, 5, 6, 10, 12, 15, 20, 30 or 60.

When N is longer than your files, each file still gives its first minute. That is why N = 10 with 5-minute files gives a clip every 5 minutes, not every 10.

## How clips are named

Many recorders name each recording with the date and time it started. Audio Splitter names each clip with the date and time **the clip** starts:

| Part of `SITE01_20260119_140500.WAV` | Meaning |
| --- | --- |
| `SITE01_` | stays as in the recording's name |
| `20260119` | date: 19 January 2026 (year, month, day) |
| `140500` | time: 14:05:00 (hours, minutes, seconds) |
| `.WAV` | the same file type as the recording |

**The last two digits are seconds:** `140005` would be 14:00:05, five seconds after 14:00, not five minutes.

A 1-hour recording named `SITE01_20260119_140000.WAV`, with N = 5, gives 12 clips:

```text
SITE01_20260119_140000.WAV
SITE01_20260119_140500.WAV
SITE01_20260119_141000.WAV
…
SITE01_20260119_145500.WAV
```

The first clip has the same name as the recording because it starts at the same moment. It is a new file in the Output Folder; the recording itself is not touched. After midnight, the date moves on to the next day.

Audio Splitter finds the date and time when they are written like `20260119_140000`: 8 digits, an underscore, then 6 digits. If a name has no date and time, the clips are numbered with the minute of the recording where they start: with N = 5, `rain_test.wav` gives `rain_test_0000m.wav`, `rain_test_0005m.wav`, `rain_test_0010m.wav` …

## Where the clips are saved

The Output Folder gets the same subfolders as the Source Folder, so the clips from each site stay apart. With N = 5:

```text
Recordings/                      ← Source Folder
├── SITE01/
│   └── 20260119_140000.WAV      (1 hour)
└── SITE02/
    └── 20260119_140000.WAV      (1 hour)

Clips/                           ← Output Folder
├── SITE01/
│   ├── 20260119_140000.WAV
│   ├── 20260119_140500.WAV
│   ├── …                        (12 clips)
│   └── 20260119_145500.WAV
└── SITE02/
    └── …                        (the same 12 names, from the SITE02 recording)
```

## When it finishes

The summary at the bottom of the window says how it went:

- **clips extracted**: new clips saved this time.
- **skipped**: clips that were already in the Output Folder from an earlier run. They are left as they are.
- **errors**: recordings that could not be read, or clips that could not be saved. The Log lists them; everything else is still done.

Its first words sum it up: **Done.** (everything worked), **Done, with errors.** (the Log lists the problems), **Cancelled.** (you stopped it; run it again to continue) or **Failed.** (nothing could be done, and the message says why).

## Stopping and running again

Click **Cancel** to stop at any time. Clips already saved are kept, and a clip is only saved once it is complete, so stopping never leaves half-made clips.

To continue later, start again with the same folders and the same N. Clips that already exist are skipped, so only the missing ones are made. To make a clip again, delete it from the Output Folder first.

## Questions

**I want one clip every 5 minutes. Should I enter 4 or 5?**\
5. N counts from the start of one clip to the start of the next, and the 1-minute clip is the first of those 5 minutes. With 4 you get one clip every 4 minutes instead.

**Will Audio Splitter change or delete my recordings?**\
No. It only reads them. The clips are new files in the Output Folder.

**I entered 10, but I got a clip every 5 minutes.**\
Your recorder probably saves 5-minute files. Each file gives at least its first minute, so you get one clip per file. See [Recordings saved in several files](#recordings-saved-in-several-files).

**Why is there no clip at the end of my recording?**\
Less than a full minute was left.

**Can clips be longer or shorter than 1 minute?**\
No. Clips are always 1 minute long.

**Which recordings does it read?**\
WAV, FLAC, MP3, OGG, M4A and OPUS. Hidden files, the `._` files Macs leave on memory cards, and system folders are ignored.

**What happens to stereo recordings and to the sound quality?**\
Clips are mono: the two channels are mixed into one. They have the same file type as the recording. WAV and FLAC clips also keep the recording's sample rate and bit depth; MP3, OGG, M4A and OPUS clips are compressed again, as when an audio editor saves them.

**The summary shows errors. What should I do?**\
Look at the Log: each error names the recording or clip. Often the recording is damaged or empty, for example when the recorder's battery ran out. All the other recordings are processed normally.

**How do I make the text bigger, or leave full screen?**\
Use **View → Zoom In** or **Zoom Out** (Ctrl and + or Ctrl and −; on a Mac, Cmd), and **View → Actual Size** to go back. In full screen (**View → Toggle Full Screen**, F11; on a Mac, Ctrl+Cmd+F), press **Esc** or click **Exit full screen** at the top right to leave it.

**On a Mac, every recording fails with `signal: abort trap`.**\
That is version 0.1.0 for Macs with an Apple chip, which cannot process recordings. Install a newer version.

**It says “Could not start ffprobe” or “Could not start ffmpeg”.**\
These are the tools inside Audio Splitter that read and cut the audio, and they could not run on this computer. Install the latest version again. If the message comes back, [report it](../../issues) and copy the whole message into the report.

**Clips made with version 0.1.0 are almost empty.**\
Version 0.1.0 could leave a clip of a fraction of a second at the end of each recording. Run Audio Splitter again with the same folders: most of them are replaced by full clips. To find the rest, sort the Output Folder by size; they are only a few KB. Delete them.

---

## For developers

The rest of this page is for people who build or change Audio Splitter. It is built with **Electron**, **Vue 3**, **TypeScript**, and a **Go** backend powered by `ffmpeg`/`ffprobe`.

## Architecture

This is a three-layer desktop app: **Vue 3 frontend → Electron main process → Go backend**.

1. The user configures parameters in the Vue UI and clicks Start.
2. The renderer sends an IPC message (`start-processing`) via the context bridge (`electron/preload.ts`).
3. The Electron main process (`electron/main.ts`) starts the Go backend on demand and forwards the request over stdin.
4. The Go backend (`backend/`) uses `ffprobe` to read each file's duration, plans the clips, and runs `ffmpeg` in a worker pool. Progress (throttled) and log lines are written to stdout as JSON.
5. Electron reads stdout line by line and forwards `progress`, `log`, and `done` events to the renderer. If the backend exits mid-job, Electron reports it to the renderer, and the next job starts a new backend.
6. The Pinia store (`src/stores/processing.ts`) consumes these events and drives `ProgressBar.vue` and `LogPanel.vue`.

Every `start` command ends with exactly one `done` event, which carries `cancelled` or `error` when the job did not complete. Closing the backend's stdin cancels the job, stops ffmpeg, and exits.

The installers bundle the Go backend and `ffmpeg`/`ffprobe` for their CPU, so users install nothing else.

### Processing rules

The user guide above describes what people see. Behind it:

- **Tools** (`backend/media.go`): before a job, `ffmpeg -version` and `ffprobe -version` must run, or the job fails with one error that names the tool and the cause, instead of failing every file.
- **Clip plan** (`backend/plan.go`): for each recording, one 60-second clip every `target_step` minutes from offset 0, as long as at least 59 seconds of audio are left. The one-second slack keeps a last clip that stops a few milliseconds short. Recorders often write files a few milliseconds longer than nominal (a "5-minute" file of 300.032 s): those do not get an extra, nearly empty clip, whose timestamped name would collide with the next recording's first clip.
- **Names**: a `YYYYMMDD_HHMMSS` timestamp anywhere in the file name is replaced with the clip's start time (recording start + offset × `record_step`). Without one, clips are named `<name>_<MMMM>m<ext>` after their offset in minutes. When two clips map to the same output path (overlapping recordings), the first is kept and the log says so.
- **Cutting**: `ffmpeg -ss <offset> -i <input> -t 60 -map 0:a:0 -ac 1`, seeking before the input. PCM WAV keeps its codec, so its bit depth; other formats use ffmpeg's default encoder for the extension (FLAC keeps its sample format). Each clip is written to a hidden `.partial` file and renamed when complete.
- **Resuming**: an existing clip of at least 256 KB counts as complete. A smaller one is probed, and made again when it is shorter than 59 seconds (earlier versions could leave 32 ms clips).
- **Scan** (`backend/scan.go`): skips hidden files and folders (including the `._*` files macOS writes on memory cards), `$RECYCLE.BIN`, `System Volume Information`, and the output folder when it is inside the source folder. Folders that cannot be read are skipped with a warning.

| Setting | Config key | Default | Range |
| --- | --- | --- | --- |
| Extract Every N Minutes | `target_step` | 10 | 1–100,000 |
| Record Step (not in the UI) | `record_step` | 1 | 1–1,000: real-world minutes per audio minute, for recorders whose clock advances faster than the audio |
| Parallel Processes | `num_threads` | 0 | 0–1,024: `0` is one per CPU core, and more than 4× the core count is capped |

The UI and the backend both reject values outside these ranges.

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

### Local build for testing

Package the app for this computer only, without publishing anything. On Linux (x64):

```bash
pnpm install
bash scripts/download-ffmpeg.sh x64   # ffmpeg/ffprobe into resources/x64/
pnpm build:backend:release            # backend binaries into dist-backend/
pnpm build                            # frontend and Electron main process

# Fastest: the unpacked app, no installer
pnpm exec electron-builder --linux dir --x64 --publish never
./release/linux-unpacked/audio-splitter --no-sandbox

# Or the installers users get
pnpm exec electron-builder --linux deb AppImage --x64 --publish never
sudo apt install ./release/audio-splitter_*_amd64.deb   # installs to /opt/Audio Splitter
./release/Audio\ Splitter-*.AppImage --no-sandbox
```

`--no-sandbox` is for local testing only: recent Ubuntu releases block Electron's sandbox for apps started from a folder or an AppImage (see [Troubleshooting](#troubleshooting)). The `.deb` sets the sandbox up when it is installed, so the installed app runs normally.

On macOS and Windows, use the matching ffmpeg script and package without signing:

```bash
# macOS (Apple Silicon; use x64 on Intel Macs)
bash scripts/download-ffmpeg-macos.sh arm64
pnpm build:backend:release && pnpm build
CSC_IDENTITY_AUTO_DISCOVERY=false pnpm exec electron-builder --mac dir --arm64 --publish never
open "release/mac-arm64/Audio Splitter.app"
```

```powershell
# Windows (PowerShell)
pwsh scripts/download-ffmpeg-windows.ps1
pnpm build:backend:release
pnpm build
pnpm exec electron-builder --win dir --x64 --publish never
& "release\win-unpacked\Audio Splitter.exe"
```

To try changes without packaging, `pnpm dev` (see [Run](#run)) is faster.

### Full build

```bash
pnpm dist
```

This builds every CPU this OS packages, like the release workflow, so it needs ffmpeg for each of them in `resources/<arch>/` (on Linux: x64, arm64 and armv7l). It runs in order:

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

## Troubleshooting

| Problem | Fix |
| --- | --- |
| Build fails with `copy-backend: ffmpeg is not in the <arch> package` | Download ffmpeg for that CPU into `resources/<arch>/` (see [ffmpeg](#ffmpeg)). `pnpm dist` packages every CPU of the OS and needs all of them. |
| Build fails with `copy-backend: …audio-splitter-backend-<arch> not found` | Run `pnpm build:backend:release`. `pnpm build:backend` only builds the development binary. |
| `.deb` build fails with `Please specify project homepage` | electron-builder takes the homepage from the git remote. When building from a copy without `.git`, add `"homepage": "https://github.com/wildmon-ai/audio-splitter"` to `package.json`. |
| App exits with `The SUID sandbox helper binary was found, but is not configured correctly` | Ubuntu's AppArmor blocks Electron's sandbox for apps started from a folder or an AppImage. For local tests, start the app with `--no-sandbox`, or install the `.deb`. |
| AppImage does not start (`libfuse.so.2` missing) | `sudo apt install libfuse2t64` (Ubuntu 24.04 and later) or `sudo apt install libfuse2`. |
| `pnpm dev` shows `Processing engine not found … Run "pnpm build:backend"` | Run `pnpm build:backend` (`pnpm dev` does it when it starts). |
| Job stops with `Could not start ffprobe (…)` (or `ffmpeg`); in 0.1.0 on a Mac, every file failed with `signal: abort trap` | The bundled tool cannot run on that computer: the install is damaged, or the tool needs libraries the computer does not have. 0.1.0 for Apple Silicon shipped Homebrew's ffmpeg, which fails on Macs without Homebrew's ffmpeg: install a later version. When packaging for macOS, get ffmpeg with `scripts/download-ffmpeg-macos.sh`, never from Homebrew. |
| Clips of a few milliseconds in older output folders | Versions up to 0.1.0 could write them at the end of each recording. Running again with the same folders re-creates the ones named like a recording's clip. The others match no recording: list small files with `find <output-folder> -type f -size -64k`, check them, and delete them. |

## Project Structure

```text
src/                        Vue 3 frontend
  components/
    FolderPicker.vue        Native folder selection
    ParameterForm.vue       Processing parameters, with validation
    ProgressBar.vue         Scan and clip progress
    LogPanel.vue            Scrollable log output
    FullScreenExit.vue      "Exit full screen" button and Esc while in full screen
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
