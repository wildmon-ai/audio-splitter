// electron-builder afterPack hook. Copies the Go backend built for this
// package's CPU (pnpm build:backend:release) into the app's resources, and
// fails the build when ffmpeg/ffprobe were not bundled, instead of shipping
// an app that cannot process anything.
const fs = require('fs')
const path = require('path')

// electron-builder's Arch enum, in order.
const ARCH_NAMES = ['ia32', 'x64', 'armv7l', 'arm64', 'universal']

exports.default = async function copyBackend(context) {
  const arch = ARCH_NAMES[context.arch]
  const platform = context.electronPlatformName
  const ext = platform === 'win32' ? '.exe' : ''

  const src = path.join(__dirname, '..', 'dist-backend', `audio-splitter-backend-${arch}${ext}`)
  if (!fs.existsSync(src)) {
    throw new Error(`copy-backend: ${src} not found. Run "pnpm build:backend:release" first (${arch} is not built on this OS).`)
  }

  const resourcesDir =
    platform === 'darwin'
      ? path.join(context.appOutDir, `${context.packager.appInfo.productFilename}.app`, 'Contents', 'Resources')
      : path.join(context.appOutDir, 'resources')

  for (const tool of ['ffmpeg', 'ffprobe']) {
    if (!fs.existsSync(path.join(resourcesDir, tool + ext))) {
      throw new Error(
        `copy-backend: ${tool}${ext} is not in the ${arch} package. Put it in resources/${arch}/ ` +
          '(see scripts/download-ffmpeg*.sh).',
      )
    }
  }

  const dest = path.join(resourcesDir, `audio-splitter-backend${ext}`)
  fs.copyFileSync(src, dest)
  if (platform !== 'win32') fs.chmodSync(dest, 0o755)
}
