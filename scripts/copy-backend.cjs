const fs = require('fs')
const path = require('path')

// electron-builder Arch enum: ia32=0, x64=1, armv7l=2, arm64=3, universal=4
const ARCH_MAP = {
  0: { goSuffix: '-x64' },
  1: { goSuffix: '-x64' },
  2: { goSuffix: '-armv7l' },
  3: { goSuffix: '-arm64' },
  4: { goSuffix: '-x64' },
}

exports.default = async function (context) {
  const platform = context.electronPlatformName
  const ext = platform === 'win32' ? '.exe' : ''

  let src
  if (platform === 'linux' || platform === 'darwin') {
    const archInfo = ARCH_MAP[context.arch] || { goSuffix: '-x64' }
    src = path.join(__dirname, '..', 'dist-backend', `audio-splitter-backend${archInfo.goSuffix}`)
  } else {
    src = path.join(__dirname, '..', 'dist-backend', `audio-splitter-backend${ext}`)
  }

  const resourcesDir =
    platform === 'darwin'
      ? path.join(context.appOutDir, `${context.packager.appInfo.productFilename}.app`, 'Contents', 'Resources')
      : path.join(context.appOutDir, 'resources')

  const dest = path.join(resourcesDir, `audio-splitter-backend${ext}`)
  if (!fs.existsSync(src)) {
    throw new Error(`copy-backend: source binary not found at ${src}`)
  }
  fs.mkdirSync(path.dirname(dest), { recursive: true })
  fs.copyFileSync(src, dest)
  if (platform !== 'win32') fs.chmodSync(dest, 0o755)
}
