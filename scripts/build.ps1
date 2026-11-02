$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'toolchain.ps1')
$cyreneGo = Use-CyreneGoToolchain
Push-Location (Split-Path -Parent $PSScriptRoot)
try {
    & $cyreneGo tool mygo build @args
    if ($LASTEXITCODE -ne 0) { throw "MyGo exited with code $LASTEXITCODE" }
} finally {
    Pop-Location
}
