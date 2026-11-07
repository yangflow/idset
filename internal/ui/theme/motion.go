package theme

import "github.com/egoist/mygo/ui"

// The motion of the screens, one function per kind of moment in design/tokens.json's motion.moments. Each says where
// something came from, what happened, or look here; MyGo keeps only the fades and color changes under reduced motion.

// shownKey is the state an element keeps of when it was first built, for Arrive.
type shownKey struct{}

// Arrive gives e its way in from enter and its way out to exit: it comes in over MotionBase and, once it has been
// shown that long, goes over MotionFast, so that leaving is quicker than arriving. e needs the same identity from
// frame to frame, a Key. Either motion may be nil. Its colors change at once; Recolor is for colors.
func Arrive(c *ui.Context, e ui.Element, enter, exit *ui.Motion) ui.Element {
	shown := ui.Local(e, shownKey{}, c.Now)
	d := MotionBase
	if shown != nil && c.Now().Sub(*shown) >= MotionBase {
		d = MotionFast
	}
	return e.Transition(ui.ElementTransition{Duration: d, Ease: Ease, Position: true, Size: true, Enter: enter, Exit: exit})
}

// FadeIn fades e in over MotionFast as it appears, and out as it goes: a word such as Copied, an editor's content.
func FadeIn(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Duration: MotionFast, Ease: Ease, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
}

// Appear fades e in over MotionFast as it appears and lets it go at once: an editor's content, for another identity,
// which replaces the last one in place. e needs a Key that changes with what it shows.
func Appear(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Duration: MotionFast, Ease: Ease, Enter: Fade()})
}

// Recolor changes e's background and border over MotionFast, and fades it in and out: the drop zone taking a folder.
func Recolor(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Duration: MotionFast, Ease: Ease, Colors: true, Enter: Fade(), Exit: Fade()})
}

// Swap cross-fades e's content over MotionFast as it changes: a copy icon that becomes a check.
func Swap(e ui.Element) ui.Element {
	return e.ContentTransition(ui.ContentTransition{Animation: &ui.Animation{Duration: MotionFast, Ease: Ease}})
}

// Turn turns e, a chevron, to degrees over MotionBase: 90 points an expanded section's chevron down.
func Turn(e ui.Element, degrees float32) ui.Element {
	return e.AnimatePose(ui.Pose{Rotate: degrees}, ui.Animation{Duration: MotionBase, Ease: Ease})
}

// Slide is a downward move by MotionShift, from above (Slide(-1)) or below (Slide(1)), fading in: a card settling
// into place, a bar rising.
func Slide(dir float32) *ui.Motion { return &ui.Motion{Y: dir * MotionShift} }

// Fade is a motion in place that only fades.
func Fade() *ui.Motion { return &ui.Motion{} }

// Collapse is a motion along the parent's direction to nothing, fading: a row opening or closing, a section expanding.
func Collapse() *ui.Motion { return &ui.Motion{Collapse: true} }
