$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    go mod vendor
    if ($LASTEXITCODE -ne 0) { throw 'go mod vendor failed' }

    foreach ($patch in @('mygo-v0.7.4-rounded-webview.patch', 'mygo-v0.7.4-frosted-overlays.patch')) {
        git apply --directory=vendor/github.com/egoist/mygo (Join-Path 'docs/patches' $patch)
        if ($LASTEXITCODE -ne 0) { throw "MyGo patch $patch no longer applies; review the SDK changes." }
    }
} finally {
    Pop-Location
}
