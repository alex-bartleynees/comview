package tui

import "go.rockorager.dev/vaxis"

type Theme struct {
	Name   string
	Colors BaseColors
}

func ThemeByName(name string) (Theme, bool) {
	for _, t := range Themes {
		if t.Name == name {
			return t, true
		}
	}
	return Theme{}, false
}

var Themes = []Theme{
	{
		Name: "Default",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0xd7, 0xde, 0xe9),
			Background: vaxis.RGBColor(0x10, 0x14, 0x19),
			Red:        vaxis.RGBColor(0xe0, 0x6c, 0x75),
			Green:      vaxis.RGBColor(0x98, 0xc3, 0x79),
			Yellow:     vaxis.RGBColor(0xe5, 0xc0, 0x7b),
			Blue:       vaxis.RGBColor(0x61, 0xaf, 0xef),
			Magenta:    vaxis.RGBColor(0xc6, 0x78, 0xdd),
			Cyan:       vaxis.RGBColor(0x56, 0xb6, 0xc2),
		},
	},
	{
		Name: "Catppuccin Mocha",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0xcd, 0xd6, 0xf4),
			Background: vaxis.RGBColor(0x1e, 0x1e, 0x2e),
			Red:        vaxis.RGBColor(0xf3, 0x8b, 0xa8),
			Green:      vaxis.RGBColor(0xa6, 0xe3, 0xa1),
			Yellow:     vaxis.RGBColor(0xf9, 0xe2, 0xaf),
			Blue:       vaxis.RGBColor(0x89, 0xb4, 0xfa),
			Magenta:    vaxis.RGBColor(0xcb, 0xa6, 0xf7),
			Cyan:       vaxis.RGBColor(0x89, 0xdc, 0xeb),
		},
	},
	{
		Name: "Catppuccin Latte",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0x4c, 0x4f, 0x69),
			Background: vaxis.RGBColor(0xef, 0xf1, 0xf5),
			Red:        vaxis.RGBColor(0xd2, 0x0f, 0x39),
			Green:      vaxis.RGBColor(0x40, 0xa0, 0x2b),
			Yellow:     vaxis.RGBColor(0xdf, 0x8e, 0x1d),
			Blue:       vaxis.RGBColor(0x1e, 0x66, 0xf5),
			Magenta:    vaxis.RGBColor(0xea, 0x76, 0xcb),
			Cyan:       vaxis.RGBColor(0x04, 0xa5, 0xe5),
		},
	},
	{
		Name: "Solarized Dark",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0x83, 0x94, 0x96),
			Background: vaxis.RGBColor(0x00, 0x2b, 0x36),
			Red:        vaxis.RGBColor(0xdc, 0x32, 0x2f),
			Green:      vaxis.RGBColor(0x85, 0x99, 0x00),
			Yellow:     vaxis.RGBColor(0xb5, 0x89, 0x00),
			Blue:       vaxis.RGBColor(0x26, 0x8b, 0xd2),
			Magenta:    vaxis.RGBColor(0xd3, 0x36, 0x82),
			Cyan:       vaxis.RGBColor(0x2a, 0xa1, 0x98),
		},
	},
	{
		Name: "Solarized Light",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0x65, 0x7b, 0x83),
			Background: vaxis.RGBColor(0xfd, 0xf6, 0xe3),
			Red:        vaxis.RGBColor(0xdc, 0x32, 0x2f),
			Green:      vaxis.RGBColor(0x85, 0x99, 0x00),
			Yellow:     vaxis.RGBColor(0xb5, 0x89, 0x00),
			Blue:       vaxis.RGBColor(0x26, 0x8b, 0xd2),
			Magenta:    vaxis.RGBColor(0xd3, 0x36, 0x82),
			Cyan:       vaxis.RGBColor(0x2a, 0xa1, 0x98),
		},
	},
	{
		Name: "Tokyo Night",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0x9a, 0xab, 0xd6),
			Background: vaxis.RGBColor(0x1a, 0x1b, 0x26),
			Red:        vaxis.RGBColor(0xf7, 0x76, 0x8e),
			Green:      vaxis.RGBColor(0x9e, 0xce, 0x6a),
			Yellow:     vaxis.RGBColor(0xe0, 0xaf, 0x68),
			Blue:       vaxis.RGBColor(0x7a, 0xa2, 0xf7),
			Magenta:    vaxis.RGBColor(0xbb, 0x9a, 0xf7),
			Cyan:       vaxis.RGBColor(0x7d, 0xcf, 0xf6),
		},
	},
	{
		Name: "Gruvbox Dark",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0xeb, 0xdb, 0xb2),
			Background: vaxis.RGBColor(0x28, 0x28, 0x28),
			Red:        vaxis.RGBColor(0xfb, 0x49, 0x34),
			Green:      vaxis.RGBColor(0xb8, 0xbb, 0x26),
			Yellow:     vaxis.RGBColor(0xfa, 0xbd, 0x2f),
			Blue:       vaxis.RGBColor(0x83, 0xa5, 0x98),
			Magenta:    vaxis.RGBColor(0xd3, 0x86, 0x9b),
			Cyan:       vaxis.RGBColor(0x8e, 0xc0, 0x7c),
		},
	},
	{
		Name: "Everforest Dark",
		Colors: BaseColors{
			Foreground: vaxis.RGBColor(0xd3, 0xc6, 0xaa),
			Background: vaxis.RGBColor(0x2d, 0x35, 0x3b),
			Red:        vaxis.RGBColor(0xe6, 0x7e, 0x80),
			Green:      vaxis.RGBColor(0xa7, 0xc0, 0x80),
			Yellow:     vaxis.RGBColor(0xdb, 0xbc, 0x7f),
			Blue:       vaxis.RGBColor(0x7f, 0xbb, 0xb3),
			Magenta:    vaxis.RGBColor(0xd6, 0x99, 0xb6),
			Cyan:       vaxis.RGBColor(0x83, 0xc0, 0x92),
		},
	},
}
