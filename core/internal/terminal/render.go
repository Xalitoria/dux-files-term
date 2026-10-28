package terminal

import (
	"encoding/json"
	"fmt"
	"image/color"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

const (
	attrBold = 1 << iota
	attrFaint
	attrItalic
	attrUnderline
	attrStrike
	attrInverse
)

type run struct {
	X    int    `json:"x"`
	W    int    `json:"w"`
	Text string `json:"t"`
	Fg   string `json:"f,omitempty"`
	Bg   string `json:"b,omitempty"`
	Attr int    `json:"a,omitempty"`

	Underline      int    `json:"u,omitempty"`
	UnderlineColor string `json:"uc,omitempty"`
	Link           string `json:"l,omitempty"`
}

type palette func(index int) color.Color

func hexColor(c color.Color, colors palette) string {
	switch v := c.(type) {
	case nil:
		return ""
	case ansi.BasicColor:
		c = colors(int(v))
	case ansi.IndexedColor:
		c = colors(int(v))
	}
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func attrsOf(style uv.Style) int {
	attrs := 0
	for _, flag := range []struct {
		uv, ours int
	}{
		{uv.AttrBold, attrBold},
		{uv.AttrFaint, attrFaint},
		{uv.AttrItalic, attrItalic},
		{uv.AttrStrikethrough, attrStrike},
		{uv.AttrReverse, attrInverse},
	} {
		if int(style.Attrs)&flag.uv != 0 {
			attrs |= flag.ours
		}
	}
	if style.Underline != uv.UnderlineNone {
		attrs |= attrUnderline
	}
	return attrs
}

func styleOf(cell *uv.Cell, colors palette) run {
	r := run{Fg: hexColor(cell.Style.Fg, colors), Bg: hexColor(cell.Style.Bg, colors), Attr: attrsOf(cell.Style)}
	if cell.Style.Underline != uv.UnderlineNone {
		r.Underline = int(cell.Style.Underline)
		r.UnderlineColor = hexColor(cell.Style.UnderlineColor, colors)
	}
	if !strings.HasPrefix(cell.Link.URL, markerPrefix) {
		r.Link = cell.Link.URL
	}
	return r
}

func sameStyle(a, b run) bool {
	return a.Fg == b.Fg && a.Bg == b.Bg && a.Attr == b.Attr && a.Underline == b.Underline && a.UnderlineColor == b.UnderlineColor && a.Link == b.Link
}

func invisible(r run) bool {
	return r.Bg == "" && r.Link == "" && r.Attr&(attrInverse|attrUnderline|attrStrike) == 0
}

// Wide and non-ASCII characters get a run of their own: they can come from a fallback font with a
// different advance, so the UI has to place each of them on the cell grid.
func rowRuns(width int, cellAt func(x int) *uv.Cell, colors palette) []run {
	runs := []run{}
	joinable := false
	for x := 0; x < width; {
		cell := cellAt(x)
		if cell == nil || cell.Content == "" {
			cell = &uv.EmptyCell
		}
		span := max(cell.Width, 1)
		style := styleOf(cell, colors)
		alone := span > 1 || !isASCII(cell.Content)
		if !joinable || !sameStyle(runs[len(runs)-1], style) || alone {
			style.X = x
			runs = append(runs, style)
		}
		runs[len(runs)-1].Text += cell.Content
		runs[len(runs)-1].W += span
		joinable = !alone
		x += span
	}
	return trimRuns(runs)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func trimRuns(runs []run) []run {
	for len(runs) > 0 {
		last := &runs[len(runs)-1]
		if !invisible(*last) {
			return runs
		}
		if trimmed := strings.TrimRight(last.Text, " "); trimmed != "" {
			last.W -= len(last.Text) - len(trimmed)
			last.Text = trimmed
			return runs
		}
		runs = runs[:len(runs)-1]
	}
	return runs
}

func encodeRow(runs []run) json.RawMessage {
	data, _ := json.Marshal(runs)
	return data
}
