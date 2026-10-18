// Package icons holds the hand-drawn glyphs of the UI: 16 x 16 line drawings that take the color of the text around
// them (ui.Icon). They are the SVGs of the mockups in design/screens/; there is no icon set or image file.
package icons

import "github.com/egoist/mygo/ui"

func svg(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="#000" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var (
	Close        = svg(`<path d="M4 4l8 8M12 4l-8 8" stroke-width="1.6"/>`)
	Copy         = svg(`<rect x="5.5" y="5.5" width="8" height="8" rx="1.5" stroke-width="1.4"/><path d="M10.5 3.5v-.5a1 1 0 0 0-1-1h-6a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h.5" stroke-width="1.4"/>`)
	Check        = svg(`<path d="M3.5 8.5l3 3 6-7" stroke-width="1.6"/>`)
	Plus         = svg(`<path d="M8 3v10M3 8h10" stroke-width="1.6"/>`)
	Minus        = svg(`<path d="M3.5 8h9" stroke-width="1.8"/>`)
	Warn         = svg(`<path d="M8 2.5l6 11H2z" stroke-width="1.6"/><path d="M8 6.5v3M8 11.6v.1" stroke-width="1.6"/>`)
	Error        = svg(`<circle cx="8" cy="8" r="6.25" stroke-width="1.5"/><path d="M8 4.9v3.8" stroke-width="1.5"/><path d="M8 11.1v.1" stroke-width="1.8"/>`)
	Folder       = svg(`<path d="M2 4.5a1 1 0 0 1 1-1h3l1.5 1.5H13a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1z" stroke-width="1.4"/>`)
	More         = svg(`<path d="M3.5 8h.01M8 8h.01M12.5 8h.01" stroke-width="2.2"/>`)
	Arrow        = svg(`<path d="M3 8h10M9.5 4.5L13 8l-3.5 3.5" stroke-width="1.5"/>`)
	Pin          = svg(`<path d="M9.6 2.4l4 4-1.5.6-2.3 2.3.2 2.4-.9.9-5-5 .9-.9 2.4.2 2.3-2.3z" stroke-width="1.4"/><path d="M5.7 10.3L2.5 13.5" stroke-width="1.4"/>`)
	Info         = svg(`<circle cx="8" cy="8" r="6.25" stroke-width="1.5"/><path d="M8 7.3v3.7" stroke-width="1.5"/><path d="M8 4.9v.1" stroke-width="1.8"/>`)
	Power        = svg(`<path d="M4.4 4.1a5.5 5.5 0 1 0 7.2 0" stroke-width="1.5"/><path d="M8 1.8v5.6" stroke-width="1.5"/>`)
	Globe        = svg(`<circle cx="8" cy="8" r="6.25" stroke-width="1.4"/><path d="M1.75 8h12.5" stroke-width="1.4"/><path d="M8 1.75c2.2 2 2.2 10.5 0 12.5M8 1.75c-2.2 2-2.2 10.5 0 12.5" stroke-width="1.4"/>`)
	Refresh      = svg(`<path d="M13.25 8a5.25 5.25 0 1 1-1.54-3.71" stroke-width="1.5"/><path d="M12.9 1.9v2.8H10.1" stroke-width="1.5"/>`)
	Terminal     = svg(`<path d="M3 4.5l4 3.5-4 3.5" stroke-width="1.5"/><path d="M8.5 12h4.5" stroke-width="1.5"/>`)
	ChevronRight = svg(`<path d="M6 4l4 4-4 4" stroke-width="1.6"/>`)
	ChevronDown  = svg(`<path d="M4 6l4 4 4-4" stroke-width="1.6"/>`)
	Sliders      = svg(`<path d="M2 4.5h7M12 4.5h2M2 11.5h2M7 11.5h7" stroke-width="1.4"/><circle cx="10.5" cy="4.5" r="1.5" stroke-width="1.4"/><circle cx="5.5" cy="11.5" r="1.5" stroke-width="1.4"/>`)
	Clock        = svg(`<path d="M2.5 8a5.5 5.5 0 1 0 1.8-4.1M2.5 2.5v3h3" stroke-width="1.4"/><path d="M8 5v3.2l2 1.3" stroke-width="1.4"/>`)
	Import       = svg(`<path d="M8 2v8M5 7l3 3 3-3" stroke-width="1.4"/><path d="M2.5 10.5v2a1 1 0 0 0 1 1h9a1 1 0 0 0 1-1v-2" stroke-width="1.4"/>`)
	Doc          = svg(`<path d="M4 2h5l3 3v9H4z" stroke-width="1.4"/><path d="M9 2v3h3M6 8.5h4M6 11h4" stroke-width="1.4"/>`)
	Restore      = svg(`<path d="M5.5 3L2.5 6l3 3" stroke-width="1.4"/><path d="M2.5 6H10a3.5 3.5 0 0 1 0 7H6.5" stroke-width="1.4"/>`)
	Trash        = svg(`<path d="M2.5 4.5h11M6 4.5V3h4v1.5M4 4.5l.6 8.5h6.8l.6-8.5" stroke-width="1.4"/>`)
	Gear         = svg(`<path d="M6.91 3.12L7.14 1.46L8.86 1.46L9.09 3.12L10.68 3.78L12.02 2.77L13.23 3.98L12.22 5.32L12.88 6.91L14.54 7.14L14.54 8.86L12.88 9.09L12.22 10.68L13.23 12.02L12.02 13.23L10.68 12.22L9.09 12.88L8.86 14.54L7.14 14.54L6.91 12.88L5.32 12.22L3.98 13.23L2.77 12.02L3.78 10.68L3.12 9.09L1.46 8.86L1.46 7.14L3.12 6.91L3.78 5.32L2.77 3.98L3.98 2.77L5.32 3.78z" stroke-width="1.5"/><circle cx="8" cy="8" r="2" stroke-width="1.5"/>`)
	Key          = svg(`<circle cx="5.5" cy="10.5" r="3" stroke-width="1.4"/><path d="M7.7 8.3L13.5 2.5M11 5l2 2M9.5 6.5l1.5 1.5" stroke-width="1.4"/>`)
	Card         = svg(`<rect x="1.5" y="3.5" width="13" height="9" rx="1.5" stroke-width="1.2"/><circle cx="5.5" cy="7" r="1.3" stroke-width="1.2"/><path d="M3.5 10.5a2 2 0 0 1 4 0M9.5 7h3M9.5 9.5h2" stroke-width="1.2"/>`)
	External     = svg(`<path d="M9 3h4v4M13 3L7.5 8.5M11.5 9.5V13h-8.5V4.5H6.5" stroke-width="1.4"/>`)
)
