package app

import "github.com/egoist/mygo/ui"

func (a *application) overlayVisible() bool {
	if a.launcher || a.palette.Open || a.preferences || a.aliasEditor || a.groupEditor || a.stopConfirm || a.resumeDialog || a.launcherMotion.Visible || a.palette.Motion.Visible {
		return true
	}
	for _, motion := range a.dialogs {
		if motion.Visible {
			return true
		}
	}
	return false
}

// Native browser layers are not part of the UI backdrop. Temporarily show a
// captured frame while a popup is visible, so the same filter can soften it.
// The live page stays loaded and returns after the overlay's exit animation.
func (a *application) syncOverlaySnapshots() bool {
	visible := a.overlayVisible()
	changed := visible != a.overlaySnapshotsActive
	if changed {
		a.overlaySnapshotsActive = visible
		a.overlaySnapshotEpoch++
		for _, session := range a.work.Tabs {
			for _, p := range session.Root.leaves() {
				if b := p.Browser; b != nil {
					b.OverlaySnapshot, b.OverlayCaptured = nil, false
				}
			}
		}
	}
	if !visible || a.win == nil {
		return changed
	}
	screen := a.work.activeScreen()
	if screen == nil {
		return changed
	}
	for _, p := range screen.Root.leaves() {
		b := p.Browser
		if b == nil || b.View == nil || b.OverlayCaptured {
			continue
		}
		b.OverlayCaptured = true
		view, win, epoch, id := b.View, a.win, a.overlaySnapshotEpoch, p.ID
		go func() {
			data, err := view.CapturePage()
			if err != nil {
				return
			}
			bitmap, err := ui.DecodeBitmap(data)
			if err != nil {
				return
			}
			win.Update(func() {
				if a.closed.Load() || a.overlaySnapshotEpoch != epoch || b.View != view || a.work.sessionForPane(id) == nil {
					return
				}
				b.OverlaySnapshot = bitmap
			})
		}()
	}
	return changed
}
