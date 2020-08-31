package dynamiccli

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/morikuni/aec"
	sshterm "golang.org/x/crypto/ssh/terminal"
)

// TODO: docs
type Document struct {
	mu          sync.Mutex
	w           io.Writer
	cols        uint
	rows        uint
	els         []Element
	refreshRate time.Duration
	lastCount   uint
}

// SetOutput sets the location where rendering will be drawn.
func (d *Document) SetOutput(w io.Writer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.w = w
}

func (d *Document) SetSize(rows, cols uint) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rows = rows
	d.cols = cols
}

func (d *Document) SetRefreshRate(dur time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refreshRate = dur
}

func (d *Document) Add(el ...Element) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.els = append(d.els, el...)
}

// Render starts a render loop that continues to render until the
// context is cancelled.
func (d *Document) Render(ctx context.Context) {
	dur := d.refreshRate
	if dur == 0 {
		dur = time.Second / 12
	}

	t := time.NewTicker(dur)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			d.RenderFrame()
		}
	}
}

// RenderFrame will render a single frame and return.
//
// If a manual size is not configured, this will recalcualte the window
// size on each call. This typically requires a syscall. This is a bit
// expensive but something we can optimize in the future if it ends up being
// a real source of FPS issues.
func (d *Document) RenderFrame() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// If we don't have a writer set, then don't render anything.
	if d.w == nil {
		return
	}

	// Detect if we had a window size change
	cols := d.cols
	rows := d.rows
	if cols == 0 || rows == 0 {
		if f, ok := d.w.(*os.File); ok && sshterm.IsTerminal(int(f.Fd())) {
			ws, err := pty.GetsizeFull(f)
			if err == nil {
				rows = uint(ws.Rows)
				cols = uint(ws.Cols)
			}
		}
	}

	// We always have one less row than the size of the window because
	// we draw a newline at the end of every render.
	// NOTE(mitchellh): This is very fixable and we probably want to one day
	rows -= 1

	// We have to double render to determine the line count of each element.
	// Then we go back and rerender so that we only clear less than the
	// number of rows in the screen. This is important to avoid an infinite
	// scrollback scenario.
	var count, offset uint
	for i := len(d.els) - 1; i >= 0; i-- {
		el := d.els[i]
		thisCount := el.Render(ioutil.Discard, cols)
		nextCount := count + thisCount
		if nextCount > rows {
			break
		}

		offset++
		count = nextCount
	}

	// Remove the correct number of lines. If we're rendering more lines now
	// than we ever have before, then we only clear what we drew before
	// otherwise we'll delete back before us.
	clear := count
	if clear > d.lastCount {
		clear = d.lastCount
	}
	fmt.Fprint(d.w, b.Up(clear).Column(0).EraseLine(aec.EraseModes.All).ANSI)

	// Go back and do our render
	for _, el := range d.els[len(d.els)-int(offset):] {
		el.Render(d.w, cols)
		fmt.Fprintln(d.w)
	}

	// Store how much we drew
	d.lastCount = count
}

var b = aec.EmptyBuilder