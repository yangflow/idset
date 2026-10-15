// Package icons holds the hand-drawn glyphs of the UI: 16 x 16 line drawings that take the color of the text around
// them (ui.Icon). They are the SVGs of design/screens/panel.html; there is no icon set or image file.
package icons

import "github.com/egoist/mygo/ui"

func svg(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="#000" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var (
	Close  = svg(`<path d="M4 4l8 8M12 4l-8 8" stroke-width="1.6"/>`)
	Copy   = svg(`<rect x="5.5" y="5.5" width="8" height="8" rx="1.5" stroke-width="1.4"/><path d="M10.5 3.5v-.5a1 1 0 0 0-1-1h-6a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h.5" stroke-width="1.4"/>`)
	Check  = svg(`<path d="M3.5 8.5l3 3 6-7" stroke-width="1.6"/>`)
	Plus   = svg(`<path d="M8 3v10M3 8h10" stroke-width="1.6"/>`)
	Minus  = svg(`<path d="M3.5 8h9" stroke-width="1.6"/>`)
	Warn   = svg(`<path d="M8 2.5l6 11H2z" stroke-width="1.6"/><path d="M8 6.5v3M8 11.6v.1" stroke-width="1.6"/>`)
	Folder = svg(`<path d="M2 4.5a1 1 0 0 1 1-1h3l1.5 1.5H13a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1z" stroke-width="1.4"/>`)
	More   = svg(`<path d="M3.5 8h.01M8 8h.01M12.5 8h.01" stroke-width="2.2"/>`)
	Arrow  = svg(`<path d="M3 8h10M9.5 4.5L13 8l-3.5 3.5" stroke-width="1.5"/>`)
	Pin    = svg(`<path d="M9.6 2.4l4 4-1.5.6-2.3 2.3.2 2.4-.9.9-5-5 .9-.9 2.4.2 2.3-2.3z" stroke-width="1.4"/><path d="M5.7 10.3L2.5 13.5" stroke-width="1.4"/>`)
	Info   = svg(`<circle cx="8" cy="8" r="6.25" stroke-width="1.5"/><path d="M8 7.3v3.7" stroke-width="1.5"/><path d="M8 4.9v.1" stroke-width="1.8"/>`)
	Power  = svg(`<path d="M4.4 4.1a5.5 5.5 0 1 0 7.2 0" stroke-width="1.5"/><path d="M8 1.8v5.6" stroke-width="1.5"/>`)
)
