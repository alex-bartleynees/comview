package tui

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

// Run starts the comview TUI.
func Run() error {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		return err
	}

	return app.Run(&appWidget{})
}

type appWidget struct{}

func (a *appWidget) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	key, ok := ev.(vaxis.Key)
	if !ok {
		return nil, nil
	}

	switch {
	case key.Matches('c', vaxis.ModCtrl),
		key.Matches('d', vaxis.ModCtrl),
		key.Matches('q'),
		key.MatchString("Esc"):
		return vxfw.QuitCmd{}, nil
	default:
		return nil, nil
	}
}

func (a *appWidget) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	switch ev.(type) {
	case vxfw.Init:
		return vxfw.SetTitleCmd("comview"), nil
	}
	return nil, nil
}

func (a *appWidget) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	surface := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, a)

	writeString(&surface, ctx, 0, 0, "comview", vaxis.Style{
		Foreground: vaxis.IndexColor(14),
		Attribute:  vaxis.AttrBold,
	})
	writeString(&surface, ctx, 0, 2, "TUI shell initialized with vaxis/vxfw.", vaxis.Style{})
	writeString(&surface, ctx, 0, 4, "Press q, Esc, Ctrl+C, or Ctrl+D to quit.", vaxis.Style{
		Foreground: vaxis.IndexColor(8),
	})

	return surface, nil
}

func writeString(surface *vxfw.Surface, ctx vxfw.DrawContext, col uint16, row uint16, text string, style vaxis.Style) {
	if row >= surface.Size.Height {
		return
	}

	for _, char := range ctx.Characters(text) {
		if col >= surface.Size.Width {
			return
		}

		surface.WriteCell(col, row, vaxis.Cell{
			Character: char,
			Style:     style,
		})
		col += uint16(char.Width)
	}
}
