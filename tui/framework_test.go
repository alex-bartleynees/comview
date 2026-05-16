package tui

import (
	"errors"
	"os/exec"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/term"
)

func TestFramePipelineRendersFirstRequestImmediately(t *testing.T) {
	pipeline := NewFramePipeline(60)
	now := time.Unix(1, 0)
	frames := 0

	pipeline.request(now, func() {
		frames++
	})

	if frames != 1 {
		t.Fatalf("frames = %d, want 1", frames)
	}
	if pipeline.pending {
		t.Fatal("first request left a pending frame")
	}
}

func TestFramePipelineCoalescesUntilNextTick(t *testing.T) {
	pipeline := NewFramePipeline(60)
	now := time.Unix(1, 0)
	frames := 0

	render := func() {
		frames++
	}

	pipeline.request(now, render)
	pipeline.request(now.Add(time.Millisecond), render)
	pipeline.request(now.Add(2*time.Millisecond), render)

	if frames != 1 {
		t.Fatalf("frames before tick = %d, want 1", frames)
	}
	if !pipeline.pending {
		t.Fatal("coalesced requests did not leave a pending frame")
	}

	pipeline.tick(now.Add(pipeline.Interval()), render)

	if frames != 2 {
		t.Fatalf("frames after tick = %d, want 2", frames)
	}
	if pipeline.pending {
		t.Fatal("tick left a pending frame")
	}
}

func TestFramePipelineRendersImmediatelyAfterInterval(t *testing.T) {
	pipeline := NewFramePipeline(60)
	now := time.Unix(1, 0)
	frames := 0

	render := func() {
		frames++
	}

	pipeline.request(now, render)
	pipeline.request(now.Add(pipeline.Interval()), render)

	if frames != 2 {
		t.Fatalf("frames = %d, want 2", frames)
	}
	if pipeline.pending {
		t.Fatal("request after interval left a pending frame")
	}
}

func TestConstraintsConstrain(t *testing.T) {
	constraints := Constraints{
		Min: Size{Width: 10, Height: 5},
		Max: Size{Width: 20, Height: 15},
	}

	tests := []struct {
		name string
		size Size
		want Size
	}{
		{
			name: "below min",
			size: Size{Width: 1, Height: 2},
			want: Size{Width: 10, Height: 5},
		},
		{
			name: "above max",
			size: Size{Width: 30, Height: 40},
			want: Size{Width: 20, Height: 15},
		},
		{
			name: "inside",
			size: Size{Width: 12, Height: 8},
			want: Size{Width: 12, Height: 8},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := constraints.Constrain(test.size); got != test.want {
				t.Fatalf("Constrain(%+v) = %+v, want %+v", test.size, got, test.want)
			}
		})
	}
}

func TestConstraintsConstrainUnboundedMax(t *testing.T) {
	constraints := Constraints{
		Min: Size{Width: 10, Height: 5},
		Max: Size{Width: Unbounded, Height: Unbounded},
	}

	got := constraints.Constrain(Size{Width: 1000, Height: 2000})
	want := Size{Width: 1000, Height: 2000}
	if got != want {
		t.Fatalf("Constrain with unbounded max = %+v, want %+v", got, want)
	}
}

func TestConstraintsConstrainPartiallyUnboundedMax(t *testing.T) {
	constraints := Constraints{
		Max: Size{Width: 20, Height: Unbounded},
	}

	got := constraints.Constrain(Size{Width: 1000, Height: 2000})
	want := Size{Width: 20, Height: 2000}
	if got != want {
		t.Fatalf("Constrain with partially unbounded max = %+v, want %+v", got, want)
	}
}

func TestUnconstrained(t *testing.T) {
	constraints := Unconstrained()
	if !constraints.Max.HasUnboundedWidth() {
		t.Fatal("Unconstrained max width is bounded")
	}
	if !constraints.Max.HasUnboundedHeight() {
		t.Fatal("Unconstrained max height is bounded")
	}
}

func TestTerminalWidgetStartsWithInitialSize(t *testing.T) {
	terminal := &fakeTerminalModel{}

	widget, err := newTerminalWidget(terminal, exec.Command("true"), Size{Width: 100, Height: 40})
	if err != nil {
		t.Fatal(err)
	}

	if widget.size != (Size{Width: 100, Height: 40}) {
		t.Fatalf("size = %+v, want 100x40", widget.size)
	}
	if !terminal.focused {
		t.Fatal("terminal was not focused")
	}
	if terminal.startWidth != 100 || terminal.startHeight != 40 {
		t.Fatalf("start size = %dx%d, want 100x40", terminal.startWidth, terminal.startHeight)
	}
}

func TestTerminalWidgetUsesDefaultSize(t *testing.T) {
	terminal := &fakeTerminalModel{}

	widget, err := newTerminalWidget(terminal, exec.Command("true"), Size{})
	if err != nil {
		t.Fatal(err)
	}

	if widget.size != (Size{Width: 80, Height: 24}) {
		t.Fatalf("size = %+v, want 80x24", widget.size)
	}
	if terminal.startWidth != 80 || terminal.startHeight != 24 {
		t.Fatalf("start size = %dx%d, want 80x24", terminal.startWidth, terminal.startHeight)
	}
}

func TestTerminalWidgetClosesOnStartError(t *testing.T) {
	wantErr := errors.New("start failed")
	terminal := &fakeTerminalModel{startErr: wantErr}

	_, err := newTerminalWidget(terminal, exec.Command("true"), Size{Width: 80, Height: 24})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if !terminal.closed {
		t.Fatal("terminal was not closed after start error")
	}
}

func TestTerminalWidgetResizesDuringLayout(t *testing.T) {
	terminal := &fakeTerminalModel{}
	widget, err := newTerminalWidget(terminal, exec.Command("true"), Size{Width: 80, Height: 24})
	if err != nil {
		t.Fatal(err)
	}

	size := widget.Layout(Tight(Size{Width: 120, Height: 50}))

	if size != (Size{Width: 120, Height: 50}) {
		t.Fatalf("layout size = %+v, want 120x50", size)
	}
	if terminal.resizeWidth != 120 || terminal.resizeHeight != 50 {
		t.Fatalf("resize = %dx%d, want 120x50", terminal.resizeWidth, terminal.resizeHeight)
	}
}

func TestTerminalWidgetRoutesInputEvents(t *testing.T) {
	terminal := &fakeTerminalModel{}
	widget, err := newTerminalWidget(terminal, exec.Command("true"), Size{Width: 80, Height: 24})
	if err != nil {
		t.Fatal(err)
	}

	key := vaxis.Key{Text: ":", Keycode: ';', ShiftedCode: ':', Modifiers: vaxis.ModShift}
	cmd, err := widget.HandleEvent(key)
	if err != nil {
		t.Fatal(err)
	}

	if cmd != CommandRedraw {
		t.Fatalf("command = %v, want %v", cmd, CommandRedraw)
	}
	if len(terminal.updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(terminal.updates))
	}
	if got, ok := terminal.updates[0].(vaxis.Key); !ok || got != key {
		t.Fatalf("update = %#v, want %#v", terminal.updates[0], key)
	}
}

func TestTerminalWidgetIgnoresHostResizeInput(t *testing.T) {
	terminal := &fakeTerminalModel{}
	widget, err := newTerminalWidget(terminal, exec.Command("true"), Size{Width: 80, Height: 24})
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := widget.HandleEvent(vaxis.Resize{Cols: 120, Rows: 50})
	if err != nil {
		t.Fatal(err)
	}

	if cmd != CommandRedraw {
		t.Fatalf("command = %v, want %v", cmd, CommandRedraw)
	}
	if len(terminal.updates) != 0 {
		t.Fatalf("updates = %d, want 0", len(terminal.updates))
	}
}

func TestTerminalWidgetClosesOnTerminalEvent(t *testing.T) {
	wantErr := errors.New("exit status 1")
	terminal := &fakeTerminalModel{}
	widget, err := newTerminalWidget(terminal, exec.Command("true"), Size{Width: 80, Height: 24})
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := widget.HandleEvent(term.EventClosed{Error: wantErr})
	if err != nil {
		t.Fatal(err)
	}

	if cmd != CommandCloseTerminal {
		t.Fatalf("command = %v, want %v", cmd, CommandCloseTerminal)
	}
	if !errors.Is(widget.Err(), wantErr) {
		t.Fatalf("terminal err = %v, want %v", widget.Err(), wantErr)
	}
	if len(terminal.updates) != 0 {
		t.Fatalf("updates = %d, want 0", len(terminal.updates))
	}
}

func TestTerminalOverlayLayout(t *testing.T) {
	layout, ok := terminalOverlayLayout(100, 40)
	if !ok {
		t.Fatal("layout missing")
	}
	if layout.boxWidth != 92 || layout.boxHeight != 36 {
		t.Fatalf("box = %dx%d, want 92x36", layout.boxWidth, layout.boxHeight)
	}
	if layout.x != 4 || layout.y != 2 {
		t.Fatalf("position = %d,%d, want 4,2", layout.x, layout.y)
	}
	if layout.innerWidth != 90 || layout.innerHeight != 34 {
		t.Fatalf("inner = %dx%d, want 90x34", layout.innerWidth, layout.innerHeight)
	}
}

func TestTranslateTerminalMouse(t *testing.T) {
	layout, ok := terminalOverlayLayout(100, 40)
	if !ok {
		t.Fatal("layout missing")
	}
	mouse := vaxis.Mouse{
		Col:    layout.x + 1 + 10,
		Row:    layout.y + 1 + 5,
		XPixel: 150,
		YPixel: 160,
	}

	got, ok := translateTerminalMouse(mouse, layout, vaxis.Resize{
		Cols:   100,
		Rows:   40,
		XPixel: 1000,
		YPixel: 800,
	})
	if !ok {
		t.Fatal("mouse was outside terminal")
	}
	if got.Col != 10 || got.Row != 5 {
		t.Fatalf("translated cell = %d,%d, want 10,5", got.Col, got.Row)
	}
	if got.XPixel != 100 || got.YPixel != 100 {
		t.Fatalf("translated pixel = %d,%d, want 100,100", got.XPixel, got.YPixel)
	}
}

func TestTranslateTerminalMouseOutside(t *testing.T) {
	layout, ok := terminalOverlayLayout(100, 40)
	if !ok {
		t.Fatal("layout missing")
	}

	_, ok = translateTerminalMouse(vaxis.Mouse{Col: layout.x, Row: layout.y}, layout, vaxis.Resize{})
	if ok {
		t.Fatal("border mouse translated as terminal input")
	}
}

func TestEditorCommandAddsLineArguments(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		target   EditorTarget
		wantName string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "default vi",
			target:   EditorTarget{Path: "main.go", Line: 12, Column: 4},
			wantName: "vi",
			wantArgs: []string{"+call cursor(12,4)", "main.go"},
		},
		{
			name:     "editor with args",
			editor:   "nvim -p",
			target:   EditorTarget{Path: "main.go", Line: 12, Column: 4},
			wantName: "nvim",
			wantArgs: []string{"-p", "+call cursor(12,4)", "main.go"},
		},
		{
			name:     "quoted editor path",
			editor:   `"/opt/My Editor/bin/nvim" --clean`,
			target:   EditorTarget{Path: "main.go", Line: 12, Column: 4},
			wantName: "/opt/My Editor/bin/nvim",
			wantArgs: []string{"--clean", "+call cursor(12,4)", "main.go"},
		},
		{
			name:     "code",
			editor:   "code --reuse-window",
			target:   EditorTarget{Path: "main.go", Line: 12, Column: 4},
			wantName: "code",
			wantArgs: []string{"--reuse-window", "-g", "main.go:12:4"},
		},
		{
			name:     "unknown editor",
			editor:   "ed",
			target:   EditorTarget{Path: "main.go", Line: 12, Column: 4},
			wantName: "ed",
			wantArgs: []string{"main.go"},
		},
		{
			name:    "bad editor command",
			editor:  `"nvim`,
			target:  EditorTarget{Path: "main.go", Line: 12, Column: 4},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args, err := editorCommand(tt.editor, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatal("err = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if name != tt.wantName {
				t.Fatalf("name = %q, want %q", name, tt.wantName)
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, tt.wantArgs)
			}
			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Fatalf("args = %#v, want %#v", args, tt.wantArgs)
				}
			}
		})
	}
}

type fakeTerminalModel struct {
	attached     bool
	focused      bool
	blurred      bool
	closed       bool
	drawn        bool
	startErr     error
	startWidth   int
	startHeight  int
	resizeWidth  int
	resizeHeight int
	updates      []vaxis.Event
}

func (f *fakeTerminalModel) Attach(func(vaxis.Event)) {
	f.attached = true
}

func (f *fakeTerminalModel) Blur() {
	f.blurred = true
}

func (f *fakeTerminalModel) Close() {
	f.closed = true
}

func (f *fakeTerminalModel) Draw(vaxis.Window) {
	f.drawn = true
}

func (f *fakeTerminalModel) Focus() {
	f.focused = true
}

func (f *fakeTerminalModel) Resize(width int, height int) {
	f.resizeWidth = width
	f.resizeHeight = height
}

func (f *fakeTerminalModel) StartWithSize(_ *exec.Cmd, width int, height int) error {
	f.startWidth = width
	f.startHeight = height
	return f.startErr
}

func (f *fakeTerminalModel) Update(ev vaxis.Event) {
	f.updates = append(f.updates, ev)
}
