// Builds the Go backend into dist-backend/.
//
//   node scripts/build-backend.cjs            this computer's OS and CPU, as
//                                             dist-backend/audio-splitter-backend[.exe] (used by `pnpm dev`)
//   node scripts/build-backend.cjs --release  every CPU this OS packages, as
//                                             dist-backend/audio-splitter-backend-<arch>[.exe]
//                                             (copied into the app by scripts/copy-backend.cjs)
const { execFileSync } = require('child_process')
const fs = require('fs')
const path = require('path')

const root = path.join(__dirname, '..')
const outDir = path.join(root, 'dist-backend')
const goCache = path.join(root, '.cache', 'go-build')

const GOOS = { win32: 'windows', darwin: 'darwin', linux: 'linux' }[process.platform]
// electron-builder / Node arch name → Go settings
const GO_ARCH = {
  x64: { GOARCH: 'amd64' },
  arm64: { GOARCH: 'arm64' },
  armv7l: { GOARCH: 'arm', GOARM: '7' },
  arm: { GOARCH: 'arm', GOARM: '7' }, // Node's name for 32-bit ARM
}
// Must match the "arch" lists of the electron-builder targets in package.json.
const RELEASE_ARCHS = { windows: ['x64'], darwin: ['x64', 'arm64'], linux: ['x64', 'arm64'] }

if (!GOOS) throw new Error(`build-backend: unsupported platform ${process.platform}`)
const ext = GOOS === 'windows' ? '.exe' : ''

function build(arch, outName) {
  const goArch = GO_ARCH[arch]
  if (!goArch) throw new Error(`build-backend: unsupported CPU ${arch}`)
  execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', path.join(outDir, outName), '.'], {
    cwd: path.join(root, 'backend'),
    env: { ...process.env, GOCACHE: process.env.GOCACHE || goCache, CGO_ENABLED: '0', GOOS, ...goArch },
    stdio: 'inherit',
  })
}

fs.mkdirSync(outDir, { recursive: true })
fs.mkdirSync(goCache, { recursive: true })

if (process.argv.includes('--release')) {
  for (const arch of RELEASE_ARCHS[GOOS]) build(arch, `audio-splitter-backend-${arch}${ext}`)
} else {
  build(process.arch, `audio-splitter-backend${ext}`)
}
