const { execFileSync } = require('child_process')
const fs = require('fs')
const path = require('path')

const root = path.join(__dirname, '..')
const outDir = path.join(root, 'dist-backend')
const goCache = path.join(root, '.cache', 'go-build')

fs.mkdirSync(outDir, { recursive: true })
fs.mkdirSync(goCache, { recursive: true })

const goEnv = { ...process.env, GOCACHE: process.env.GOCACHE || goCache }

function build(goarch, goos, outName) {
  const dest = path.join(outDir, outName)
  execFileSync('go', ['build', '-o', dest, '.'], {
    cwd: path.join(root, 'backend'),
    env: { ...goEnv, GOOS: goos, GOARCH: goarch },
    stdio: 'inherit',
  })
}

const isWindows = process.platform === 'win32'
const isMac = process.platform === 'darwin'

if (isWindows) {
  build('amd64', 'windows', 'audio-splitter-backend.exe')
} else if (isMac) {
  // build both archs so electron-builder can package x64 and arm64 DMGs from one runner
  build('amd64', 'darwin', 'audio-splitter-backend-x64')
  build('arm64', 'darwin', 'audio-splitter-backend-arm64')
} else {
  // Linux CI: build for all Linux architectures
  build('amd64', 'linux', 'audio-splitter-backend-x64')
  build('arm64', 'linux', 'audio-splitter-backend-arm64')
  build('arm', 'linux', 'audio-splitter-backend-armv7l')
}
