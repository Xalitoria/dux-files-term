package terminal

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

const (
	maxTransferBytes = 64 << 20
	maxImagePixels   = 40_000_000
	maxStoredBytes   = 64 << 20
	maxPlacements    = 4096
	maxAPCBytes      = 96 << 20
	markerPrefix     = "dfimg:"
	autoIDBase       = 1 << 30
	defaultCellW     = 10
	defaultCellH     = 20
)

var (
	apcStart = []byte("\x1b_G")
	apcEnd   = []byte("\x1b\\")
)

type storedImage struct {
	uid    int
	id     int
	number int
	png    []byte
	w, h   int
	used   int
	sent   bool
}

type placement struct {
	serial         int
	image          *storedImage
	pid            int
	cols, rows     int
	sx, sy, sw, sh int
	ox, oy         int
	z              int
	w, h           float64
}

type placementView struct {
	Image int     `json:"i"`
	X     int     `json:"x"`
	Y     int     `json:"y"`
	Cols  int     `json:"c"`
	Rows  int     `json:"r"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	SX    int     `json:"sx"`
	SY    int     `json:"sy"`
	SW    int     `json:"sw"`
	SH    int     `json:"sh"`
	OX    int     `json:"ox,omitempty"`
	OY    int     `json:"oy,omitempty"`
	Z     int     `json:"z,omitempty"`
}

type frameImage struct {
	ID  int    `json:"id"`
	Src string `json:"src"`
}

type command struct {
	keys    map[byte]string
	payload []byte
}

func (c command) number(key byte) int {
	n, _ := strconv.Atoi(c.keys[key])
	return n
}

func (c command) letter(key, fallback byte) byte {
	if v := c.keys[key]; v != "" {
		return v[0]
	}
	return fallback
}

func parseCommand(raw []byte) command {
	head, payload, _ := bytes.Cut(raw, []byte(";"))
	c := command{keys: map[byte]string{}, payload: payload}
	for _, pair := range strings.Split(string(head), ",") {
		key, value, ok := strings.Cut(pair, "=")
		if ok && len(key) == 1 {
			c.keys[key[0]] = value
		}
	}
	return c
}

type transfer struct {
	cmd  command
	data []byte
}

type graphics struct {
	emu     *vt.Emulator
	reply   func([]byte)
	cell    func() (int, int)
	images  map[int]*storedImage
	places  map[int]*placement
	pending *transfer
	dropped []int
	last    string
	stored  int
	tick    int
	nextUID int
	nextSer int
	nextID  int

	stream apcStream
}

func newGraphics(emu *vt.Emulator, reply func([]byte), cell func() (int, int)) *graphics {
	return &graphics{
		emu:     emu,
		reply:   reply,
		cell:    cell,
		images:  map[int]*storedImage{},
		places:  map[int]*placement{},
		nextID:  autoIDBase,
		nextUID: 1,
		nextSer: 1,
	}
}

// apcStream splits kitty graphics commands out of the output so they never reach the emulator.
type apcStream struct {
	buf    []byte
	carry  []byte
	inside bool
	skip   bool
}

func (g *graphics) feed(data []byte) {
	g.stream.feed(data, func(p []byte) { _, _ = g.emu.Write(p) }, g.handle)
}

func (a *apcStream) feed(data []byte, plain func([]byte), command func([]byte)) {
	if len(a.carry) > 0 {
		data = append(a.carry, data...)
		a.carry = nil
	}
	for len(data) > 0 {
		switch {
		case a.skip:
			data = a.discard(data)
		case a.inside:
			data = a.collect(data, command)
		default:
			data = a.scan(data, plain)
		}
	}
}

func (a *apcStream) scan(data []byte, plain func([]byte)) []byte {
	i := bytes.Index(data, apcStart)
	if i < 0 {
		keep := partialStart(data)
		plain(data[:len(data)-keep])
		a.carry = bytes.Clone(data[len(data)-keep:])
		return nil
	}
	plain(data[:i])
	a.inside = true
	return data[i+len(apcStart):]
}

func partialStart(data []byte) int {
	for keep := min(len(apcStart)-1, len(data)); keep > 0; keep-- {
		if bytes.HasPrefix(apcStart, data[len(data)-keep:]) {
			return keep
		}
	}
	return 0
}

func (a *apcStream) collect(data []byte, command func([]byte)) []byte {
	from := max(len(a.buf)-1, 0)
	a.buf = append(a.buf, data...)
	end := bytes.Index(a.buf[from:], apcEnd)
	switch {
	case end >= 0:
		end += from
		command(a.buf[:end])
		rest := bytes.Clone(a.buf[end+len(apcEnd):])
		a.reset()
		return rest
	case len(a.buf) > maxAPCBytes:
		a.buf = a.buf[len(a.buf)-1:]
		a.skip = true
	}
	return nil
}

func (a *apcStream) discard(data []byte) []byte {
	joined := append(a.buf, data...)
	end := bytes.Index(joined, apcEnd)
	if end < 0 {
		a.buf = bytes.Clone(joined[max(len(joined)-1, 0):])
		return nil
	}
	rest := bytes.Clone(joined[end+len(apcEnd):])
	a.reset()
	return rest
}

func (a *apcStream) reset() {
	a.inside, a.skip = false, false
	if cap(a.buf) > 1<<20 {
		a.buf = nil
		return
	}
	a.buf = a.buf[:0]
}

func (g *graphics) handle(raw []byte) {
	c := parseCommand(raw)
	if g.pending != nil {
		g.continueTransfer(c)
		return
	}
	switch c.letter('a', 't') {
	case 't', 'T', 'q':
		g.beginTransfer(c)
	case 'p':
		g.put(c)
	case 'd':
		g.remove(c)
	default:
		g.respond(c, 0, fail("ENOSYS", "unsupported action"))
	}
}

func (g *graphics) beginTransfer(c command) {
	g.pending = &transfer{cmd: c}
	g.continueTransfer(command{keys: map[byte]string{'m': c.keys['m']}, payload: c.payload})
}

func (g *graphics) continueTransfer(c command) {
	p := g.pending
	chunk, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(string(c.payload), "="))
	switch {
	case err != nil:
		err = fail("EINVAL", "%v", err)
	case len(p.data)+len(chunk) > maxTransferBytes:
		err = fail("EFBIG", "image too large")
	}
	if err != nil {
		g.pending = nil
		g.respond(p.cmd, 0, err)
		return
	}
	p.data = append(p.data, chunk...)
	if c.keys['m'] == "1" {
		return
	}
	g.pending = nil
	g.transmit(p.cmd, p.data)
}

func fail(code, format string, args ...any) error {
	return fmt.Errorf("%s:%s", code, fmt.Sprintf(format, args...))
}

func encodeJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}

func (g *graphics) transmit(c command, data []byte) {
	data, err := g.load(c, data)
	var png []byte
	var w, h int
	if err == nil {
		png, w, h, err = encodeImage(c, data)
	}
	if err != nil {
		g.respond(c, 0, err)
		return
	}
	if c.letter('a', 't') == 'q' {
		g.respond(c, 0, nil)
		return
	}
	img := g.store(c, png, w, h)
	g.respond(c, img.id, nil)
	if c.letter('a', 't') == 'T' {
		g.place(c, img)
	}
}

func (g *graphics) load(c command, data []byte) ([]byte, error) {
	data, err := readMedium(c, data)
	if err != nil {
		return nil, err
	}
	if c.letter('o', 0) != 'z' {
		return data, nil
	}
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fail("EINVAL", "%v", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(io.LimitReader(zr, maxTransferBytes+1))
	switch {
	case err != nil:
		return nil, fail("EINVAL", "%v", err)
	case len(out) > maxTransferBytes:
		return nil, fail("EFBIG", "image too large")
	}
	return out, nil
}

func readMedium(c command, data []byte) ([]byte, error) {
	medium := c.letter('t', 'd')
	if medium == 'd' {
		return data, nil
	}
	path := string(data)
	if medium == 's' {
		name := strings.TrimPrefix(path, "/")
		if name == "" || strings.Contains(name, "/") || name == ".." {
			return nil, fail("EINVAL", "bad shared memory name")
		}
		path = "/dev/shm/" + name
	}
	if !filepath.IsAbs(path) || (medium != 's' && hasAnyPrefix(path, "/proc/", "/sys/", "/dev/")) {
		return nil, fail("EINVAL", "unsupported path")
	}
	out, err := readRegularFile(path, c.number('O'), c.number('S'))
	if err != nil {
		return nil, fail("ENOENT", "%v", err)
	}
	if medium == 's' || (medium == 't' && strings.Contains(path, "tty-graphics-protocol") && hasAnyPrefix(path, os.TempDir()+"/", "/tmp/", "/dev/shm/")) {
		_ = os.Remove(path)
	}
	return out, nil
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	return slices.ContainsFunc(prefixes, func(prefix string) bool { return strings.HasPrefix(s, prefix) })
}

func readRegularFile(path string, offset, size int) ([]byte, error) {
	info, err := os.Stat(path)
	switch {
	case err != nil:
		return nil, err
	case !info.Mode().IsRegular():
		return nil, errors.New("not a regular file")
	case info.Size() > maxTransferBytes:
		return nil, errors.New("file too large")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if offset > 0 {
		if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
			return nil, err
		}
	}
	reader := io.Reader(file)
	if size > 0 {
		reader = io.LimitReader(file, int64(size))
	}
	return io.ReadAll(reader)
}

func encodeImage(c command, data []byte) ([]byte, int, int, error) {
	format := c.number('f')
	if format == 0 {
		format = 32
	}
	if format == 100 {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		switch {
		case err != nil:
			return nil, 0, 0, fail("EBADPNG", "cannot decode the image")
		case cfg.Width*cfg.Height > maxImagePixels:
			return nil, 0, 0, fail("EFBIG", "image too large")
		}
		return data, cfg.Width, cfg.Height, nil
	}
	w, h := c.number('s'), c.number('v')
	bytesPer := map[int]int{24: 3, 32: 4}[format]
	switch {
	case bytesPer == 0:
		return nil, 0, 0, fail("EINVAL", "unsupported format")
	case w <= 0 || h <= 0 || w*h > maxImagePixels:
		return nil, 0, 0, fail("EINVAL", "bad image size")
	case len(data) != w*h*bytesPer:
		return nil, 0, 0, fail("EINVAL", "data does not match the size")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	switch bytesPer {
	case 4:
		copy(img.Pix, data)
	default:
		for i := range w * h {
			copy(img.Pix[i*4:], data[i*3:i*3+3])
			img.Pix[i*4+3] = 255
		}
	}
	var out bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&out, img); err != nil {
		return nil, 0, 0, fail("EINVAL", "%v", err)
	}
	return out.Bytes(), w, h, nil
}

func (g *graphics) store(c command, data []byte, w, h int) *storedImage {
	id, number := c.number('i'), c.number('I')
	if id == 0 && number > 0 {
		g.nextID++
		id = g.nextID
	}
	if old := g.imageByID(id); old != nil {
		g.free(old)
	}
	g.tick++
	img := &storedImage{uid: g.nextUID, id: id, number: number, png: data, w: w, h: h, used: g.tick}
	g.nextUID++
	g.images[img.uid] = img
	g.stored += len(data)
	g.evict(img)
	return img
}

func (g *graphics) evict(keep *storedImage) {
	for g.stored > maxStoredBytes {
		var oldest *storedImage
		for _, img := range g.images {
			if img != keep && (oldest == nil || img.used < oldest.used) {
				oldest = img
			}
		}
		if oldest == nil {
			return
		}
		g.free(oldest)
	}
}

func (g *graphics) imageByID(id int) *storedImage {
	if id == 0 {
		return nil
	}
	for _, img := range g.images {
		if img.id == id {
			return img
		}
	}
	return nil
}

func (g *graphics) imageByNumber(number int) *storedImage {
	var newest *storedImage
	for _, img := range g.images {
		if img.number == number && (newest == nil || img.uid > newest.uid) {
			newest = img
		}
	}
	return newest
}

func (g *graphics) free(img *storedImage) {
	delete(g.images, img.uid)
	g.stored -= len(img.png)
	g.dropped = append(g.dropped, img.uid)
	for serial, p := range g.places {
		if p.image == img {
			delete(g.places, serial)
		}
	}
}

func (g *graphics) respond(c command, assigned int, err error) {
	id, number, pid := c.number('i'), c.number('I'), c.number('p')
	quiet := c.number('q')
	switch {
	case id == 0 && number == 0, quiet == 2, err == nil && quiet == 1:
		return
	}
	if id == 0 {
		id = assigned
	}
	var head []string
	for _, field := range []struct {
		key   string
		value int
	}{{"i", id}, {"I", number}, {"p", pid}} {
		if field.value > 0 {
			head = append(head, field.key+"="+strconv.Itoa(field.value))
		}
	}
	message := "OK"
	if err != nil {
		message = err.Error()
	}
	g.reply([]byte("\x1b_G" + strings.Join(head, ",") + ";" + message + "\x1b\\"))
}

func (g *graphics) put(c command) {
	img := g.imageByID(c.number('i'))
	if img == nil && c.number('i') == 0 {
		img = g.imageByNumber(c.number('I'))
	}
	if img == nil {
		g.respond(c, 0, fail("ENOENT", "no such image"))
		return
	}
	g.respond(c, img.id, nil)
	g.place(c, img)
}

func (g *graphics) place(c command, img *storedImage) {
	if c.keys['U'] == "1" {
		return
	}
	p := g.geometry(c, img)
	if pid := c.number('p'); pid > 0 {
		g.removeSerials(g.serialsWhere(func(o *placement) bool { return o.image == img && o.pid == pid }))
	}
	p.serial = g.nextSer
	g.nextSer++
	g.tick++
	img.used = g.tick
	g.places[p.serial] = p
	g.trimPlacements()

	x := min(g.emu.CursorPosition().X, g.emu.Width()-1)
	_, _ = g.emu.WriteString(strings.Repeat("\n", p.rows-1))
	g.mark(p, x, g.emu.CursorPosition().Y-(p.rows-1))
	if c.number('C') == 1 {
		_, _ = g.emu.WriteString(fmt.Sprintf("\x1b[%dA\x1b[%dG", p.rows-1, x+1))
		return
	}
	_, _ = g.emu.WriteString(fmt.Sprintf("\x1b[%dC", p.cols))
}

func (g *graphics) geometry(c command, img *storedImage) *placement {
	cw, ch := g.cell()
	if cw <= 0 || ch <= 0 {
		cw, ch = defaultCellW, defaultCellH
	}
	p := &placement{image: img, pid: c.number('p'), ox: min(max(c.number('X'), 0), cw-1), oy: min(max(c.number('Y'), 0), ch-1), z: c.number('z')}
	p.sx, p.sy = min(max(c.number('x'), 0), img.w-1), min(max(c.number('y'), 0), img.h-1)
	p.sw, p.sh = c.number('w'), c.number('h')
	if p.sw <= 0 || p.sx+p.sw > img.w {
		p.sw = img.w - p.sx
	}
	if p.sh <= 0 || p.sy+p.sh > img.h {
		p.sh = img.h - p.sy
	}
	cols, rows := c.number('c'), c.number('r')
	wide, tall := float64(cols*cw-p.ox), float64(rows*ch-p.oy)
	switch {
	case cols <= 0 && rows <= 0:
		wide, tall = float64(p.sw), float64(p.sh)
	case rows <= 0:
		tall = wide * float64(p.sh) / float64(p.sw)
	case cols <= 0:
		wide = tall * float64(p.sw) / float64(p.sh)
	}
	p.cols, p.rows = max(1, ceilDiv(int(wide)+p.ox, cw)), max(1, ceilDiv(int(tall)+p.oy, ch))
	if height := g.emu.Height(); p.rows > height {
		scale := float64(height) / float64(p.rows)
		wide, tall = wide*scale, tall*scale
		p.cols, p.rows = max(1, ceilDiv(int(wide)+p.ox, cw)), height
	}
	p.cols = min(p.cols, maxCols)
	p.w, p.h = wide/float64(cw), tall/float64(ch)
	return p
}

func ceilDiv(a, b int) int {
	return (a + b - 1) / b
}

func (g *graphics) trimPlacements() {
	for len(g.places) > maxPlacements {
		oldest := -1
		for serial := range g.places {
			if oldest < 0 || serial < oldest {
				oldest = serial
			}
		}
		delete(g.places, oldest)
	}
}

func markerOf(serial, dx, dy int) string {
	return markerPrefix + strconv.Itoa(serial) + ":" + strconv.Itoa(dx) + ":" + strconv.Itoa(dy)
}

func parseMarker(url string) (serial, dx, dy int, ok bool) {
	rest, found := strings.CutPrefix(url, markerPrefix)
	if !found {
		return 0, 0, 0, false
	}
	parts := strings.Split(rest, ":")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var err error
	nums := [3]int{}
	for i, part := range parts {
		if nums[i], err = strconv.Atoi(part); err != nil {
			return 0, 0, 0, false
		}
	}
	return nums[0], nums[1], nums[2], true
}

func (g *graphics) mark(p *placement, x, y int) {
	width := g.emu.Width()
	for dy := range p.rows {
		for dx := 0; dx < p.cols && x+dx < width; dx++ {
			g.setLink(x+dx, y+dy, markerOf(p.serial, dx, dy))
		}
	}
}

func (g *graphics) setLink(x, y int, url string) {
	cell := g.emu.CellAt(x, y)
	if cell == nil {
		return
	}
	next := cell.Clone()
	next.Link = uv.Link{URL: url}
	g.emu.SetCell(x, y, next)
}

type located struct {
	serial int
	x, y   int
}

// Global line indexes count scrollback first, then the screen rows.
func (g *graphics) history() int {
	if g.emu.IsAltScreen() {
		return 0
	}
	return g.emu.ScrollbackLen()
}

func (g *graphics) eachMarker(from, to int, visit func(line, x int, link *uv.Link) *uv.Link) {
	history, width := g.history(), g.emu.Width()
	for i := max(from, 0); i < to && i < history+g.emu.Height(); i++ {
		if i < history {
			g.visitLine(g.emu.Scrollback().Line(i), i, visit)
			continue
		}
		for x := range width {
			cell := g.emu.CellAt(x, i-history)
			if cell == nil || cell.Link.URL == "" {
				continue
			}
			if next := visit(i, x, &cell.Link); next != nil {
				g.setLink(x, i-history, next.URL)
			}
		}
	}
}

func (g *graphics) visitLine(line uv.Line, index int, visit func(line, x int, link *uv.Link) *uv.Link) {
	for x := range line {
		if line[x].Link.URL == "" {
			continue
		}
		if next := visit(index, x, &line[x].Link); next != nil {
			line[x].Link = *next
		}
	}
}

func (g *graphics) locate(from, to int) []located {
	if len(g.places) == 0 {
		return nil
	}
	seen := map[int]bool{}
	var found []located
	g.eachMarker(from, to, func(line, x int, link *uv.Link) *uv.Link {
		serial, dx, dy, ok := parseMarker(link.URL)
		if !ok || seen[serial] || g.places[serial] == nil {
			return nil
		}
		seen[serial] = true
		found = append(found, located{serial: serial, x: x - dx, y: line - dy})
		return nil
	})
	slices.SortFunc(found, func(a, b located) int { return a.serial - b.serial })
	return found
}

func (g *graphics) clearMarkers(serials map[int]bool) {
	g.eachMarker(0, g.history()+g.emu.Height(), func(line, x int, link *uv.Link) *uv.Link {
		serial, _, _, ok := parseMarker(link.URL)
		if !ok || !serials[serial] {
			return nil
		}
		return &uv.Link{}
	})
}

func (g *graphics) serialsWhere(match func(*placement) bool) []int {
	var serials []int
	for serial, p := range g.places {
		if match(p) {
			serials = append(serials, serial)
		}
	}
	return serials
}

func (g *graphics) removeSerials(serials []int) {
	if len(serials) == 0 {
		return
	}
	set := make(map[int]bool, len(serials))
	for _, serial := range serials {
		set[serial] = true
		delete(g.places, serial)
	}
	g.clearMarkers(set)
}

// What d=... selects: the area of the visible placements, the image, or a cell.
func (g *graphics) remove(c command) {
	target := c.letter('d', 'a')
	free := target >= 'A' && target <= 'Z'
	lower := target | 0x20
	history := g.history()
	visible := g.locate(history, history+g.emu.Height())
	at := func(match func(l located, p *placement) bool) []int {
		var serials []int
		for _, l := range visible {
			if p := g.places[l.serial]; p != nil && match(l, p) {
				serials = append(serials, l.serial)
			}
		}
		return serials
	}
	covers := func(l located, p *placement, x, y int) bool {
		return x >= l.x && x < l.x+p.cols && y >= l.y-history && y < l.y-history+p.rows
	}
	var serials []int
	switch lower {
	case 'a':
		serials = at(func(located, *placement) bool { return true })
	case 'i':
		id, pid := c.number('i'), c.number('p')
		serials = g.serialsWhere(func(p *placement) bool { return p.image.id == id && (pid == 0 || p.pid == pid) })
	case 'n':
		number, pid := c.number('I'), c.number('p')
		serials = g.serialsWhere(func(p *placement) bool { return p.image.number == number && (pid == 0 || p.pid == pid) })
	case 'c':
		cursor := g.emu.CursorPosition()
		serials = at(func(l located, p *placement) bool { return covers(l, p, cursor.X, cursor.Y) })
	case 'p':
		serials = at(func(l located, p *placement) bool { return covers(l, p, c.number('x')-1, c.number('y')-1) })
	case 'q':
		serials = at(func(l located, p *placement) bool {
			return covers(l, p, c.number('x')-1, c.number('y')-1) && p.z == c.number('z')
		})
	case 'x':
		serials = at(func(l located, p *placement) bool { return c.number('x')-1 >= l.x && c.number('x')-1 < l.x+p.cols })
	case 'y':
		serials = at(func(l located, p *placement) bool {
			row := c.number('y') - 1
			return row >= l.y-history && row < l.y-history+p.rows
		})
	case 'z':
		serials = at(func(_ located, p *placement) bool { return p.z == c.number('z') })
	case 'r':
		lo, hi := c.number('x'), c.number('y')
		serials = g.serialsWhere(func(p *placement) bool { return p.image.id >= lo && p.image.id <= hi })
	default:
		return
	}
	affected := map[*storedImage]bool{}
	for _, serial := range serials {
		affected[g.places[serial].image] = true
	}
	g.removeSerials(serials)
	if !free {
		return
	}
	for img := range affected {
		if !g.referenced(img) {
			g.free(img)
		}
	}
}

func (g *graphics) referenced(img *storedImage) bool {
	for _, p := range g.places {
		if p.image == img {
			return true
		}
	}
	return false
}

// resync makes the next frame carry every placement and image again for a view that reconnected.
func (g *graphics) resync() {
	g.last = ""
	for _, img := range g.images {
		img.sent = false
	}
}

func (g *graphics) active() bool {
	return len(g.places) > 0 || g.last != "" || len(g.dropped) > 0
}

type graphicsFrame struct {
	Images     []frameImage
	Dropped    []int
	Placements []byte
}

// top is the global line index shown on the first row of the view.
func (g *graphics) frame(top, rows int) (graphicsFrame, bool) {
	var out graphicsFrame
	if !g.active() {
		return out, false
	}
	views := []placementView{}
	for _, l := range g.locate(top-g.emu.Height(), top+rows) {
		p := g.places[l.serial]
		if l.y+p.rows <= top || l.y >= top+rows {
			continue
		}
		views = append(views, placementView{Image: p.image.uid, X: l.x, Y: l.y - top, Cols: p.cols, Rows: p.rows, W: p.w, H: p.h, SX: p.sx, SY: p.sy, SW: p.sw, SH: p.sh, OX: p.ox, OY: p.oy, Z: p.z})
		if !p.image.sent {
			p.image.sent = true
			out.Images = append(out.Images, frameImage{ID: p.image.uid, Src: "data:image/png;base64," + base64.StdEncoding.EncodeToString(p.image.png)})
		}
	}
	out.Dropped, g.dropped = g.dropped, nil
	shown := ""
	if len(views) > 0 {
		shown = string(encodeJSON(views))
	}
	if shown != g.last {
		out.Placements = encodeJSON(views)
		g.last = shown
	}
	return out, out.Placements != nil || len(out.Images) > 0 || len(out.Dropped) > 0
}
