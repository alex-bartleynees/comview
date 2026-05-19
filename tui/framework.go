package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/term"
)

const (
	DefaultFrameRate = 60
	Unbounded        = -1
)

type Size struct {
	Width  int
	Height int
}

func (s Size) HasUnboundedWidth() bool {
	return s.Width == Unbounded
}

func (s Size) HasUnboundedHeight() bool {
	return s.Height == Unbounded
}

type Constraints struct {
	Min Size
	Max Size
}

func Tight(size Size) Constraints {
	return Constraints{
		Min: size,
		Max: size,
	}
}

func Loose(max Size) Constraints {
	return Constraints{
		Max: max,
	}
}

func Unconstrained() Constraints {
	return Loose(Size{
		Width:  Unbounded,
		Height: Unbounded,
	})
}

func (c Constraints) Constrain(size Size) Size {
	if size.Width < c.Min.Width {
		size.Width = c.Min.Width
	}
	if size.Height < c.Min.Height {
		size.Height = c.Min.Height
	}
	if !c.Max.HasUnboundedWidth() && size.Width > c.Max.Width {
		size.Width = c.Max.Width
	}
	if !c.Max.HasUnboundedHeight() && size.Height > c.Max.Height {
		size.Height = c.Max.Height
	}
	return size
}

type Command int

const (
	CommandNone Command = iota
	CommandRedraw
	CommandQuit
	CommandCopy
	CommandOpenEditor
	CommandCloseTerminal
)

type Widget interface {
	HandleEvent(vaxis.Event) (Command, error)
	Layout(Constraints) Size
	Paint(vaxis.Window)
}

type ClipboardProvider interface {
	ClipboardText() string
}

type ClipboardConsumer interface {
	ClipboardConsumed()
}

type EditorTarget struct {
	Path   string
	Line   int
	Column int
}

type EditorTargetProvider interface {
	EditorTarget() (EditorTarget, bool)
}

type StatusMessenger interface {
	SetStatusMessage(string)
}

type YankHighlighter interface {
	HighlightYank(time.Time)
	YankHighlightDuration() time.Duration
}

type TimedRedrawer interface {
	RedrawAfter() (time.Duration, bool)
}

type App struct {
	vx         *vaxis.Vaxis
	root       Widget
	terminal   *TerminalWidget
	windowSize vaxis.Resize
	frames     FramePipeline
	closeHooks []func()
}

func NewApp(root Widget, opts vaxis.Options) (*App, error) {
	vx, err := vaxis.New(opts)
	if err != nil {
		return nil, err
	}

	return NewAppWithVaxis(root, vx), nil
}

func NewAppWithVaxis(root Widget, vx *vaxis.Vaxis) *App {
	return &App{
		vx:     vx,
		root:   root,
		frames: NewFramePipeline(DefaultFrameRate),
	}
}

func (a *App) Vaxis() *vaxis.Vaxis {
	return a.vx
}

func (a *App) OnClose(fn func()) {
	a.closeHooks = append(a.closeHooks, fn)
}

func (a *App) Run() error {
	defer func() {
		for _, hook := range a.closeHooks {
			hook()
		}
		if a.terminal != nil {
			a.terminal.Close()
		}
		a.vx.Close()
	}()

	a.vx.SetTitle("comview")
	a.applyTerminalColors()
	a.frames.Request(a.draw)

	ticker := time.NewTicker(a.frames.Interval())
	defer ticker.Stop()

	events := a.vx.Events()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			cmd, err := a.handleEvent(ev)
			if err != nil {
				return err
			}
			if cmd == CommandQuit {
				return nil
			}
		case <-ticker.C:
			a.frames.Tick(a.draw)
		}
	}
}

func (a *App) draw() {
	win := a.vx.Window()
	width, height := win.Size()
	size := a.root.Layout(Tight(Size{Width: width, Height: height}))

	win.Clear()
	a.root.Paint(win.New(0, 0, size.Width, size.Height))
	a.paintTerminalOverlay(win)
	a.vx.Render()
}

func (a *App) handleEvent(ev vaxis.Event) (Command, error) {
	requestFrame := false
	switch ev := ev.(type) {
	case vaxis.Resize:
		a.windowSize = ev
		requestFrame = true
	case vaxis.Redraw:
		requestFrame = true
	case vaxis.ColorThemeUpdate:
		a.applyTerminalColors()
		requestFrame = true
	}

	if a.terminal != nil {
		cmd, err := a.handleTerminalEvent(ev)
		if err != nil {
			return CommandNone, err
		}
		if cmd == CommandCloseTerminal {
			requestFrame = true
		}
		if cmd == CommandRedraw {
			requestFrame = true
		}
		if requestFrame {
			a.frames.Request(a.draw)
			a.scheduleTimedRedraw()
		}
		return cmd, nil
	}

	cmd, err := a.root.HandleEvent(ev)
	if err != nil {
		return CommandNone, err
	}

	if cmd == CommandCopy {
		if provider, ok := a.root.(ClipboardProvider); ok {
			if text := provider.ClipboardText(); text != "" {
				a.vx.ClipboardPush(text)
			}
		}
		if highlighter, ok := a.root.(YankHighlighter); ok {
			duration := highlighter.YankHighlightDuration()
			highlighter.HighlightYank(time.Now())
			go func() {
				time.Sleep(duration)
				a.vx.PostEvent(vaxis.Redraw{})
			}()
		}
		if consumer, ok := a.root.(ClipboardConsumer); ok {
			consumer.ClipboardConsumed()
		}
		requestFrame = true
	}
	if cmd == CommandOpenEditor {
		requestFrame = true
		if err := a.openEditorTerminal(); err != nil {
			if messenger, ok := a.root.(StatusMessenger); ok {
				messenger.SetStatusMessage(fmt.Sprintf("Could not open editor: %v", err))
			} else {
				return CommandNone, err
			}
		}
	}
	if cmd == CommandRedraw {
		requestFrame = true
	}
	if requestFrame {
		a.frames.Request(a.draw)
		a.scheduleTimedRedraw()
	}
	return cmd, nil
}

func (a *App) handleTerminalEvent(ev vaxis.Event) (Command, error) {
	switch ev := ev.(type) {
	case term.EventNotify:
		a.vx.Notify(ev.Title, ev.Body)
	case term.EventMouseShape:
		a.vx.SetMouseShape(ev.Shape)
	}

	ev = a.translateTerminalEvent(ev)
	cmd, err := a.terminal.HandleEvent(ev)
	if err != nil {
		return CommandNone, err
	}
	if cmd == CommandCloseTerminal {
		terminalErr := a.terminal.Err()
		a.terminal.Close()
		a.terminal = nil
		a.vx.SetMouseShape(vaxis.MouseShapeDefault)
		a.applyTerminalColors()
		if terminalErr != nil {
			if messenger, ok := a.root.(StatusMessenger); ok {
				messenger.SetStatusMessage(fmt.Sprintf("Editor exited: %v", terminalErr))
			}
		}
	}
	return cmd, nil
}

func (a *App) translateTerminalEvent(ev vaxis.Event) vaxis.Event {
	mouse, ok := ev.(vaxis.Mouse)
	if !ok {
		return ev
	}
	width, height := a.vx.Window().Size()
	layout, ok := terminalOverlayLayout(width, height)
	if !ok {
		return mouse
	}
	resize := a.windowSize
	if resize.Cols <= 0 {
		resize.Cols = width
	}
	if resize.Rows <= 0 {
		resize.Rows = height
	}
	mouse, ok = translateTerminalMouse(mouse, layout, resize)
	if !ok {
		return vaxis.Redraw{}
	}
	return mouse
}

func (a *App) paintTerminalOverlay(win vaxis.Window) {
	if a.terminal == nil {
		return
	}
	width, height := win.Size()
	layout, ok := terminalOverlayLayout(width, height)
	if !ok {
		return
	}

	style := vaxis.Style{}
	borderStyle := vaxis.Style{Attribute: vaxis.AttrBold}
	paintBox(win, layout.x, layout.y, layout.boxWidth, layout.boxHeight, style, borderStyle)
	a.terminal.Layout(Tight(Size{Width: layout.innerWidth, Height: layout.innerHeight}))
	a.terminal.Paint(win.New(layout.x+1, layout.y+1, layout.innerWidth, layout.innerHeight))
}

func (a *App) openEditorTerminal() error {
	provider, ok := a.root.(EditorTargetProvider)
	if !ok {
		return nil
	}
	target, ok := provider.EditorTarget()
	if !ok {
		return nil
	}
	if target.Path == "" {
		return nil
	}

	name, args, err := editorCommand(configuredEditor(), target)
	if err != nil {
		return err
	}
	width, height := a.vx.Window().Size()
	layout, ok := terminalOverlayLayout(width, height)
	size := Size{Width: width, Height: height}
	if ok {
		size = Size{Width: layout.innerWidth, Height: layout.innerHeight}
	}
	terminal, err := NewTerminalWidget(a.vx, exec.Command(name, args...), size)
	if err != nil {
		return err
	}
	a.terminal = terminal
	return nil
}

func configuredEditor() string {
	if editor := strings.TrimSpace(os.Getenv("GIT_EDITOR")); editor != "" {
		return editor
	}
	if editor, ok := gitEditor(); ok {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}

func gitEditor() (string, bool) {
	output, err := exec.Command("git", "var", "GIT_EDITOR").Output()
	if err != nil {
		return "", false
	}
	editor := strings.TrimSpace(string(output))
	return editor, editor != ""
}

func editorCommand(editor string, target EditorTarget) (string, []string, error) {
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	parts, err := splitCommandLine(editor)
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 {
		parts = []string{"vi"}
	}

	line := target.Line
	if line <= 0 {
		line = 1
	}
	column := target.Column
	if column <= 0 {
		column = 1
	}
	name := parts[0]
	args := append([]string{}, parts[1:]...)
	switch filepath.Base(name) {
	case "vi", "vim", "nvim", "view", "nano", "emacs", "emacsclient":
		args = append(args, fmt.Sprintf("+call cursor(%d,%d)", line, column), target.Path)
	case "code", "code-insiders", "codium", "vscodium":
		args = append(args, "-g", fmt.Sprintf("%s:%d:%d", target.Path, line, column))
	default:
		args = append(args, target.Path)
	}
	return name, args, nil
}

func splitCommandLine(command string) ([]string, error) {
	var fields []string
	var current strings.Builder
	quote := rune(0)
	escaped := false
	haveField := false

	for _, r := range command {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
			haveField = true
		case r == '\\' && quote != '\'':
			escaped = true
			haveField = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			haveField = true
		case r == '\'' || r == '"':
			quote = r
			haveField = true
		case unicode.IsSpace(r):
			if haveField {
				fields = append(fields, current.String())
				current.Reset()
				haveField = false
			}
		default:
			current.WriteRune(r)
			haveField = true
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in editor command")
	}
	if haveField {
		fields = append(fields, current.String())
	}
	return fields, nil
}

func (a *App) scheduleTimedRedraw() {
	redrawer, ok := a.root.(TimedRedrawer)
	if !ok {
		return
	}
	duration, ok := redrawer.RedrawAfter()
	if !ok {
		return
	}
	go func() {
		time.Sleep(duration)
		a.vx.PostEvent(vaxis.Redraw{})
	}()
}

func (a *App) applyTerminalColors() {
	if receiver, ok := a.root.(TerminalColorReceiver); ok {
		receiver.SetTerminalColors(QueryTerminalColors(a.vx))
	}
}

type terminalOverlayGeometry struct {
	x           int
	y           int
	boxWidth    int
	boxHeight   int
	innerWidth  int
	innerHeight int
}

func terminalOverlayLayout(width int, height int) (terminalOverlayGeometry, bool) {
	if width < 6 || height < 5 {
		return terminalOverlayGeometry{}, false
	}

	boxWidth := width - 8
	if boxWidth < 24 {
		boxWidth = width
	}
	boxHeight := height - 4
	if boxHeight < 8 {
		boxHeight = height
	}
	if boxWidth < 3 || boxHeight < 3 {
		return terminalOverlayGeometry{}, false
	}

	x := (width - boxWidth) / 2
	y := (height - boxHeight) / 2
	return terminalOverlayGeometry{
		x:           x,
		y:           y,
		boxWidth:    boxWidth,
		boxHeight:   boxHeight,
		innerWidth:  boxWidth - 2,
		innerHeight: boxHeight - 2,
	}, true
}

func translateTerminalMouse(mouse vaxis.Mouse, layout terminalOverlayGeometry, resize vaxis.Resize) (vaxis.Mouse, bool) {
	innerX := layout.x + 1
	innerY := layout.y + 1
	if mouse.Col < innerX || mouse.Col >= innerX+layout.innerWidth ||
		mouse.Row < innerY || mouse.Row >= innerY+layout.innerHeight {
		return mouse, false
	}
	mouse.Col -= innerX
	mouse.Row -= innerY

	if mouse.XPixel > 0 && resize.XPixel > 0 && resize.Cols > 0 {
		cellWidth := resize.XPixel / resize.Cols
		if cellWidth > 0 {
			mouse.XPixel -= innerX * cellWidth
			if mouse.XPixel < 0 {
				mouse.XPixel = 0
			}
		}
	}
	if mouse.YPixel > 0 && resize.YPixel > 0 && resize.Rows > 0 {
		cellHeight := resize.YPixel / resize.Rows
		if cellHeight > 0 {
			mouse.YPixel -= innerY * cellHeight
			if mouse.YPixel < 0 {
				mouse.YPixel = 0
			}
		}
	}
	return mouse, true
}

func paintBox(win vaxis.Window, x int, y int, width int, height int, style vaxis.Style, borderStyle vaxis.Style) {
	if width <= 0 || height <= 0 {
		return
	}
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			grapheme := " "
			cellStyle := style
			switch {
			case row == y && col == x:
				grapheme = "╭"
				cellStyle = borderStyle
			case row == y && col == x+width-1:
				grapheme = "╮"
				cellStyle = borderStyle
			case row == y+height-1 && col == x:
				grapheme = "╰"
				cellStyle = borderStyle
			case row == y+height-1 && col == x+width-1:
				grapheme = "╯"
				cellStyle = borderStyle
			case row == y || row == y+height-1:
				grapheme = "─"
				cellStyle = borderStyle
			case col == x || col == x+width-1:
				grapheme = "│"
				cellStyle = borderStyle
			}
			win.SetCell(col, row, vaxis.Cell{
				Character: vaxis.Character{Grapheme: grapheme, Width: 1},
				Style:     cellStyle,
			})
		}
	}
}

type terminalModel interface {
	Attach(func(vaxis.Event))
	Blur()
	Close()
	Draw(vaxis.Window)
	Focus()
	Resize(int, int)
	StartWithSize(*exec.Cmd, int, int) error
	Update(vaxis.Event)
}

type TerminalWidget struct {
	terminal terminalModel
	size     Size
	err      error
	closed   bool
}

func NewTerminalWidget(vx *vaxis.Vaxis, cmd *exec.Cmd, size Size) (*TerminalWidget, error) {
	terminal := term.New(term.WithVaxis(vx))
	if vx != nil {
		terminal.Attach(vx.PostEvent)
	}
	return newTerminalWidget(terminal, cmd, size)
}

func newTerminalWidget(terminal terminalModel, cmd *exec.Cmd, size Size) (*TerminalWidget, error) {
	if size.Width <= 0 {
		size.Width = 80
	}
	if size.Height <= 0 {
		size.Height = 24
	}

	widget := &TerminalWidget{
		terminal: terminal,
		size:     size,
	}
	terminal.Focus()
	if err := terminal.StartWithSize(cmd, size.Width, size.Height); err != nil {
		terminal.Close()
		return nil, err
	}
	return widget, nil
}

func (w *TerminalWidget) HandleEvent(ev vaxis.Event) (Command, error) {
	switch ev := ev.(type) {
	case term.EventClosed:
		w.err = ev.Error
		w.closed = true
		return CommandCloseTerminal, nil
	case vaxis.Redraw, vaxis.Resize, term.EventNotify, term.EventMouseShape:
		return CommandRedraw, nil
	}
	w.terminal.Update(ev)
	return CommandRedraw, nil
}

func (w *TerminalWidget) Layout(constraints Constraints) Size {
	size := w.size
	if !constraints.Max.HasUnboundedWidth() {
		size.Width = constraints.Max.Width
	}
	if !constraints.Max.HasUnboundedHeight() {
		size.Height = constraints.Max.Height
	}
	size = constraints.Constrain(size)
	if size != w.size {
		w.size = size
		w.terminal.Resize(size.Width, size.Height)
	}
	return size
}

func (w *TerminalWidget) Paint(win vaxis.Window) {
	w.terminal.Draw(win)
}

func (w *TerminalWidget) Close() {
	w.terminal.Blur()
	w.terminal.Close()
	w.closed = true
}

func (w *TerminalWidget) Err() error {
	return w.err
}

type FramePipeline struct {
	interval time.Duration
	last     time.Time
	pending  bool
}

func NewFramePipeline(rate int) FramePipeline {
	if rate <= 0 {
		rate = DefaultFrameRate
	}

	return FramePipeline{
		interval: time.Second / time.Duration(rate),
	}
}

func (p *FramePipeline) Interval() time.Duration {
	return p.interval
}

func (p *FramePipeline) Request(render func()) {
	p.request(time.Now(), render)
}

func (p *FramePipeline) request(now time.Time, render func()) {
	if p.last.IsZero() || now.Sub(p.last) >= p.interval {
		p.render(now, render)
		return
	}

	p.pending = true
}

func (p *FramePipeline) Tick(render func()) {
	p.tick(time.Now(), render)
}

func (p *FramePipeline) tick(now time.Time, render func()) {
	if !p.pending {
		return
	}

	p.render(now, render)
}

func (p *FramePipeline) render(now time.Time, render func()) {
	p.last = now
	p.pending = false
	render()
}
