function Use-CyreneGoToolchain {
    $cyreneGoCommand = Get-Command go -ErrorAction SilentlyContinue
    if ($cyreneGoCommand) {
        return $cyreneGoCommand.Source
    }
    $cyrenePortableGo = Join-Path $env:LOCALAPPDATA 'cyrene-terminal\tools\go\bin\go.exe'
    if (Test-Path -LiteralPath $cyrenePortableGo) {
        $env:Path = (Split-Path -Parent $cyrenePortableGo) + ';' + $env:Path
        return $cyrenePortableGo
    }
    throw 'Go 1.27.3 or later is required. Install Go and add it to PATH.'
}
