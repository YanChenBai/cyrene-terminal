# Optional Yazi ANSI image preview fallback. Does not change system/user PATH.
$ErrorActionPreference = 'Stop'
$cyreneChafaDir = Join-Path $env:LOCALAPPDATA 'cyrene-terminal\tools\chafa'
$cyreneChafaZip = Join-Path $env:TEMP 'cyrene-chafa-1.18.3.zip'
New-Item -ItemType Directory -Force -Path $cyreneChafaDir | Out-Null
Invoke-WebRequest 'https://hpjansson.org/chafa/releases/static/chafa-1.18.3-1-x86_64-windows.zip' -OutFile $cyreneChafaZip
Expand-Archive -LiteralPath $cyreneChafaZip -DestinationPath $cyreneChafaDir -Force
Get-ChildItem -LiteralPath $cyreneChafaDir -Recurse -Filter Chafa.exe | ForEach-Object { & $_.FullName --version }
