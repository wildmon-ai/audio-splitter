# Downloads ffmpeg/ffprobe for 64-bit Windows (BtbN/FFmpeg-Builds, release
# branch, GPL) into resources\x64\, checked against the release's checksums.
# Uses the latest build of one ffmpeg release branch; when BtbN stops
# building it, the download fails and $Branch must be raised.
#
# Usage: pwsh scripts/download-ffmpeg-windows.ps1
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue' # the progress bar makes downloads much slower

$Branch = '9.0'
$Name = "ffmpeg-n$Branch-latest-win64-gpl-$Branch"
$Release = 'https://github.com/BtbN/FFmpeg-Builds/releases/download/latest'

Set-Location (Join-Path $PSScriptRoot '..')
$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
New-Item -ItemType Directory -Path $Tmp | Out-Null
try {
  $Zip = Join-Path $Tmp "$Name.zip"
  Invoke-WebRequest "$Release/$Name.zip" -OutFile $Zip
  $Sums = Join-Path $Tmp 'checksums.sha256'
  Invoke-WebRequest "$Release/checksums.sha256" -OutFile $Sums
  $Line = Get-Content $Sums | Where-Object { $_ -match "\s\*?$([regex]::Escape("$Name.zip"))\s*$" }
  if (-not $Line) { throw "No checksum for $Name.zip" }
  $Expected = ($Line -split '\s+')[0].ToLower()
  $Actual = (Get-FileHash $Zip -Algorithm SHA256).Hash.ToLower()
  if ($Expected -ne $Actual) { throw "Checksum mismatch for $Name.zip (expected $Expected, got $Actual)" }

  Expand-Archive $Zip -DestinationPath $Tmp
  New-Item -ItemType Directory -Force -Path resources\x64 | Out-Null
  Copy-Item (Join-Path $Tmp "$Name\bin\ffmpeg.exe") resources\x64\ffmpeg.exe
  Copy-Item (Join-Path $Tmp "$Name\bin\ffprobe.exe") resources\x64\ffprobe.exe
  Write-Output "Done: ffmpeg.exe and ffprobe.exe in resources\x64\"
} finally {
  Remove-Item -Recurse -Force $Tmp
}
