# cyrene-terminal

Native Go application using MyGo 0.7.4. No WebView, HTML, JavaScript, Vite or Vite+.

Follow `.agents/skills/mygo-maintenance/SKILL.md`. Read the pinned module source before using APIs. Never store `ui.Element` or capture `*ui.Context` in workers; keep state on the UI thread and publish work with `Window.Update`.

Commands: `go tool mygo dev`, `go tool mygo build -skip-installer`, `go tool mygo vet .`, `CGO_ENABLED=0 go test ./...`.

Closing the window hides the whole application to the tray and keeps terminals running. There is no separate detached/background-session feature. Closing a terminal tab requires confirmation and then ends its process. The tray context menu is the explicit application exit route. Keep starting processes and other slow I/O out of view construction.

The UI is a single immersive terminal with a 32-DIP title bar. Do not expose split controls. Collapse the sidebar completely to zero width, keeping its toggle on the title bar's left. Prefer a session alias; otherwise display the current working directory basename and update it when the directory changes. OSC title is only a fallback when no directory is available. Group-specific launch buttons inherit the group's bound directory; global launch buttons use the user's home directory. Launchers are popovers below their buttons. Modals originate at the clicked control; modals and popovers use restrained spring overshoot (at most about 1.03) and settle to 1.0, with animated exit and interrupted-transition continuity. Buttons animate their hover colors in both directions. Persist folder bindings with settings, and preserve existing sessions when editing or removing a group.

Sidebar surface spans the title bar and full window height without a divider. Support edge dragging to resize it; hide creation controls when collapsed, retaining the new-terminal shortcut. Group headers have a chevron and text separated by 7 DIPs, with the child guide centered on the chevron. Do not show folder or terminal-item icons in the sidebar. Agent logos belong only in the launcher; preserve multicolor SVGs with ui.Image rather than ui.Icon. Tab close buttons have no separate hover background. Modal and menu motion uses a continuous damped spring, not chained scale keyframes.

Project layout: `cmd/cyrene-terminal` is the entry point configured by `mygo.json`; implementation and tests live in `internal/app`. Embedded branding lives in `assets`; distribution files live in `resources`. Run all commands from the repository root. Windows wrappers are `scripts/dev.ps1` and `scripts/build.ps1`. Build acceptance binaries with `go build -tags e2e -o .mygo/cyrene-isolated-smoke.exe ./cmd/cyrene-terminal`.

The right sidebar is now a read-only file tree, using native CollapsibleBase for animated folder expansion. Git and diff views are deferred and their implementation is removed. Match the left sidebar's background without dividers. Directory reads run off the UI thread; reject stale results after a session-directory change. File clicks only select; never open, edit, delete or stage files. Use the unmodified vscode-icons SVG assets with their CC BY-SA attribution retained.
