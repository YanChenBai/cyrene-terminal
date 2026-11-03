package app

// Wrap only this process's prompt, after the user's profile has loaded.
// Keep the user's prompt and publish local filesystem locations as OSC 7.
const powerShellDirectoryIntegration = `$global:__CyreneOriginalPrompt = $function:prompt
function global:prompt {
 if ($PWD.Provider.Name -eq 'FileSystem') {
  $cyreneUri = [System.Uri]::new($PWD.ProviderPath).AbsoluteUri
  [Console]::Write(([char]27).ToString() + ']7;' + $cyreneUri + [char]7)
 }
 if ($global:__CyreneOriginalPrompt) { & $global:__CyreneOriginalPrompt } else { "PS $($PWD)> " }
}`
