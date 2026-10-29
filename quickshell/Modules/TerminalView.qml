pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Shapes
import Quickshell
import qs.Common
import qs.Services

Rectangle {
    id: term

    required property string directory

    property string sessionId: ""
    property string topic: ""
    property bool follows: false
    property int seq: 0
    property int cols: 0
    property int rows: 0
    property var colors: ({})
    property var behavior: ({})
    property int mouse: 0
    property int scrolled: 0
    property var selStart: null
    property var selEnd: null
    property var selAnchor: null
    property int clicks: 0
    property real lastPress: 0
    property int lastPressCell: -1
    property bool painted: false
    property var lineStore: []
    property var images: ({})
    property var placements: []
    property var cursor: ({
            "x": 0,
            "y": 0,
            "visible": false
        })
    property real wheelRemainder: 0
    property bool blinkVisible: true
    property bool linkHover: false
    readonly property var life: ({
            "alive": true
        })

    readonly property real cellWidth: Math.max(1, glyph.advanceWidth)
    readonly property real cellHeight: Math.max(1, Math.ceil(metrics.height))
    readonly property int fitCols: Math.floor((width - 2 * FilesMetrics.terminalPadding) / cellWidth)
    readonly property int fitRows: Math.floor((height - 2 * FilesMetrics.terminalPadding) / cellHeight)

    signal exited
    signal failed(var response)
    signal lineUpdated(int y, var runs)

    readonly property color paper: colors.bg ? colors.bg : Theme.foregroundColor(Theme.cardSurface, true)
    readonly property color ink: colors.fg ? colors.fg : Theme.surfaceText
    readonly property color cursorColor: colors.cursor ? colors.cursor : Theme.primary
    readonly property real paperOpacity: colors.opacity ?? 1
    readonly property bool customSelection: !!(colors.selectionBg || colors.selectionFg)
    readonly property color selectionPaper: colors.selectionBg ? colors.selectionBg : ink
    readonly property string cursorShape: cursor.shape || behavior.cursorShape || "block"
    readonly property bool cursorBlinks: cursor.shape ? !!cursor.blink : !!behavior.cursorBlink
    readonly property int baseFontSize: behavior.fontSize > 0 ? Math.round(behavior.fontSize * 4 / 3) : Theme.fontSizeMedium

    radius: paperOpacity < 1 ? 0 : FilesMetrics.paneRadius
    color: Qt.rgba(paper.r, paper.g, paper.b, paperOpacity)
    border.width: activeFocus ? Theme.outlineWidthFocused : 0
    border.color: Theme.primary
    clip: true
    activeFocusOnTab: true

    Component.onCompleted: open()

    Component.onDestruction: {
        life.alive = false;
        close();
    }

    onDirectoryChanged: follow()
    onActiveFocusChanged: request("terminal.focus", {
        "focused": activeFocus
    })
    onCursorChanged: {
        blinkVisible = true;
        if (blinkTimer.running)
            blinkTimer.restart();
    }
    onFitColsChanged: resizeDebounce.restart()
    onFitRowsChanged: resizeDebounce.restart()
    onCellWidthChanged: resizeDebounce.restart()
    onCellHeightChanged: resizeDebounce.restart()

    function open() {
        const life = term.life;
        DuxFilesService.call("terminal.open", {
            "cwd": directory,
            "cols": fitCols,
            "rows": fitRows
        }, result => {
            if (!life.alive) {
                if (!result.error)
                    DuxFilesService.call("terminal.close", {
                        "id": result.id
                    }, null);
                return;
            }
            if (result.error) {
                term.failed(result);
                term.exited();
                return;
            }
            term.sessionId = result.id;
            term.topic = result.topic;
            term.follows = result.follow === true;
            term.colors = result.colors ?? {};
            term.behavior = result.behavior ?? {};
            DuxFilesService.subscribeTopic(term.topic, term.handleFrame);
            term.redraw();
        });
    }

    function close() {
        if (sessionId === "")
            return;
        DuxFilesService.unsubscribeTopic(topic, handleFrame);
        DuxFilesService.call("terminal.close", {
            "id": sessionId
        }, null);
        sessionId = "";
    }

    function request(method, params) {
        if (sessionId === "")
            return;
        params.id = sessionId;
        DuxFilesService.call(method, params, null);
    }

    function redraw() {
        request("terminal.resize", {
            "cols": fitCols,
            "rows": fitRows,
            "cellWidth": Math.round(cellWidth),
            "cellHeight": cellHeight
        });
    }

    function follow() {
        if (!follows)
            return;
        request("terminal.cd", {
            "path": directory
        });
    }

    function sendInput(params) {
        if (sessionId === "")
            return;
        if (params.mouse === undefined)
            clearSelection();
        params.seq = seq++;
        request("terminal.input", params);
    }

    function handleFrame(data) {
        if (data.exited) {
            DuxFilesService.unsubscribeTopic(topic, handleFrame);
            sessionId = "";
            exited();
            return;
        }
        if (data.scroll !== scrolled) {
            scrolled = data.scroll;
            clearSelection();
        }
        if (data.rows !== rows || data.cols !== cols) {
            clearSelection();
            lineStore = Array.from({
                "length": data.rows
            }, () => []);
            cols = data.cols;
            rows = data.rows;
        }
        const lines = data.lines ?? [];
        for (const line of lines) {
            lineStore[line.y] = line.runs;
            lineUpdated(line.y, line.runs);
        }
        updateImages(data);
        if (data.clipboard !== undefined)
            Quickshell.clipboardText = data.clipboard;
        cursor = data.cursor;
        mouse = data.mouse ?? 0;
        if (painted)
            return;
        painted = lines.length === rows;
        if (!painted)
            redraw();
    }

    function updateImages(data) {
        if (data.images?.length || data.dropped?.length) {
            const next = Object.assign({}, images);
            for (const image of data.images ?? [])
                next[image.id] = image.src;
            for (const id of data.dropped ?? [])
                delete next[id];
            images = next;
        }
        if (data.placements !== undefined)
            placements = data.placements;
    }

    function linkAt(cell) {
        for (const run of lineStore[cell.y] ?? []) {
            if (run.l && cell.x >= run.x && cell.x < run.x + run.w)
                return run.l;
        }
        const text = rowCells(cell.y).join("");
        for (const match of text.matchAll(/(https?|ftp|file):\/\/[^\s<>"'`]+/g)) {
            if (cell.x >= match.index && cell.x < match.index + match[0].length)
                return match[0].replace(/[.,;:!?)\]}]+$/, "");
        }
        return "";
    }

    function openLink(url) {
        if (/^(https?|ftp|file|mailto):/i.test(url))
            DuxFilesService.openUri(url);
    }

    function rowCells(y) {
        const cells = Array.from({
            "length": cols
        }, () => " ");
        for (const run of lineStore[y] ?? []) {
            if (/^[\x00-\x7f]*$/.test(run.t)) {
                for (let i = 0; i < run.w; i++)
                    cells[run.x + i] = run.t[i];
                continue;
            }
            cells[run.x] = run.t;
            for (let i = 1; i < run.w; i++)
                cells[run.x + i] = "";
        }
        return cells;
    }

    function isWordCell(text) {
        switch (true) {
        case text === "" || /^\s$/.test(text):
            return false;
        case !!behavior.wordDelims:
            return !behavior.wordDelims.includes(text);
        case !!behavior.wordChars:
            return /^\w$/.test(text) || text.charCodeAt(0) > 127 || behavior.wordChars.includes(text);
        }
        return /^[\w@\-.\/~?&=%+#]$/.test(text);
    }

    function select(from, to) {
        const forward = from.y < to.y || (from.y === to.y && from.x <= to.x);
        selStart = forward ? from : to;
        selEnd = forward ? to : from;
    }

    function clearSelection() {
        if (selStart === null && selAnchor === null)
            return;
        selStart = null;
        selEnd = null;
        selAnchor = null;
    }

    function selectWord(cell) {
        const cells = rowCells(cell.y);
        let first = cell.x;
        let last = cell.x;
        if (isWordCell(cells[cell.x])) {
            while (first > 0 && isWordCell(cells[first - 1]))
                first--;
            while (last < cols - 1 && isWordCell(cells[last + 1]))
                last++;
        }
        selAnchor = Qt.point(first, cell.y);
        select(selAnchor, Qt.point(last, cell.y));
    }

    function selectLine(cell) {
        selAnchor = Qt.point(0, cell.y);
        select(selAnchor, Qt.point(cols - 1, cell.y));
    }

    function pressSelection(cell) {
        const now = Date.now();
        const index = cell.y * cols + cell.x;
        clicks = now - lastPress < Qt.styleHints.mouseDoubleClickInterval && index === lastPressCell ? clicks % 3 + 1 : 1;
        lastPress = now;
        lastPressCell = index;
        switch (clicks) {
        case 2:
            selectWord(cell);
            return;
        case 3:
            selectLine(cell);
            return;
        }
        clearSelection();
        selAnchor = cell;
    }

    function dragSelection(cell) {
        if (selAnchor !== null)
            select(selAnchor, cell);
    }

    function selectedText() {
        if (selStart === null)
            return "";
        const lines = [];
        for (let y = selStart.y; y <= selEnd.y; y++) {
            const from = y === selStart.y ? selStart.x : 0;
            const to = y === selEnd.y ? selEnd.x : cols - 1;
            lines.push(rowCells(y).slice(from, to + 1).join("").replace(/\s+$/, ""));
        }
        return lines.join("\n");
    }

    function copySelection() {
        const text = selectedText();
        if (text !== "")
            Quickshell.clipboardText = text;
    }

    function cellAt(point) {
        return Qt.point(Math.max(0, Math.min(cols - 1, Math.floor(point.x / cellWidth))), Math.max(0, Math.min(rows - 1, Math.floor(point.y / cellHeight))));
    }

    function buttonName(button) {
        switch (button) {
        case Qt.LeftButton:
            return "left";
        case Qt.MiddleButton:
            return "middle";
        case Qt.RightButton:
            return "right";
        }
        return "none";
    }

    function heldButton(buttons) {
        if (buttons & Qt.LeftButton)
            return "left";
        if (buttons & Qt.MiddleButton)
            return "middle";
        return buttons & Qt.RightButton ? "right" : "none";
    }

    function reportMouse(kind, cell, button, modifiers) {
        sendInput({
            "mouse": kind,
            "x": cell.x,
            "y": cell.y,
            "button": button,
            "mods": modifierMask(modifiers)
        });
    }

    function keyName(key) {
        switch (key) {
        case Qt.Key_Up:
            return "up";
        case Qt.Key_Down:
            return "down";
        case Qt.Key_Left:
            return "left";
        case Qt.Key_Right:
            return "right";
        case Qt.Key_Home:
            return "home";
        case Qt.Key_End:
            return "end";
        case Qt.Key_Insert:
            return "insert";
        case Qt.Key_Delete:
            return "delete";
        case Qt.Key_PageUp:
            return "pageUp";
        case Qt.Key_PageDown:
            return "pageDown";
        case Qt.Key_Return:
        case Qt.Key_Enter:
            return "enter";
        case Qt.Key_Tab:
        case Qt.Key_Backtab:
            return "tab";
        case Qt.Key_Backspace:
            return "backspace";
        case Qt.Key_Escape:
            return "escape";
        case Qt.Key_Space:
            return "space";
        }
        if (key >= Qt.Key_F1 && key <= Qt.Key_F12)
            return "f" + (key - Qt.Key_F1 + 1);
        return "";
    }

    function controlText(key) {
        switch (key) {
        case Qt.Key_BracketLeft:
            return "\x1b";
        case Qt.Key_Backslash:
            return "\x1c";
        case Qt.Key_BracketRight:
            return "\x1d";
        }
        if (key >= Qt.Key_A && key <= Qt.Key_Z)
            return String.fromCharCode(key - Qt.Key_A + 1);
        return "";
    }

    function modifierMask(modifiers) {
        return ((modifiers & Qt.ShiftModifier) ? 1 : 0) | ((modifiers & Qt.AltModifier) ? 2 : 0) | ((modifiers & Qt.ControlModifier) ? 4 : 0);
    }

    function setZoom(value) {
        SessionData.terminalZoom = Math.max(FilesMetrics.terminalMinZoom, Math.min(FilesMetrics.terminalMaxZoom, value));
    }

    function chordKey(key) {
        switch (true) {
        case key >= Qt.Key_A && key <= Qt.Key_Z:
            return String.fromCharCode(97 + key - Qt.Key_A);
        case key >= Qt.Key_0 && key <= Qt.Key_9:
            return String(key - Qt.Key_0);
        case key === Qt.Key_Plus || key === Qt.Key_Equal:
            return "plus";
        case key === Qt.Key_Minus || key === Qt.Key_Underscore:
            return "minus";
        }
        return keyName(key).toLowerCase();
    }

    function chordOf(event) {
        const key = chordKey(event.key);
        if (key === "")
            return "";
        const mods = event.modifiers;
        return [(mods & Qt.ControlModifier) ? "ctrl+" : "", (mods & Qt.AltModifier) ? "alt+" : "", (mods & Qt.ShiftModifier) ? "shift+" : "", (mods & Qt.MetaModifier) ? "super+" : "", key].join("");
    }

    function actionFor(chord) {
        const defaults = {
            "copy": ["ctrl+shift+c"],
            "paste": ["ctrl+shift+v"],
            "zoomIn": ["ctrl+shift+plus"],
            "zoomOut": ["ctrl+shift+minus"],
            "zoomReset": ["ctrl+shift+backspace"]
        };
        const custom = behavior.keys ?? {};
        for (const action of Object.keys(defaults)) {
            if (defaults[action].includes(chord) || (custom[action] ?? []).includes(chord))
                return action;
        }
        return "";
    }

    function runAction(action, chord) {
        switch (action) {
        case "copy":
            if (selStart === null && chord !== "ctrl+shift+c")
                return false;
            copySelection();
            return true;
        case "paste":
            sendInput({
                "text": Quickshell.clipboardText,
                "paste": true
            });
            return true;
        case "zoomIn":
            setZoom(SessionData.terminalZoom + 1);
            return true;
        case "zoomOut":
            setZoom(SessionData.terminalZoom - 1);
            return true;
        case "zoomReset":
            setZoom(0);
            return true;
        }
        return false;
    }

    function handleKey(event) {
        const chord = chordOf(event);
        if (runAction(actionFor(chord), chord))
            return;
        const mods = modifierMask(event.modifiers);
        const name = keyName(event.key);
        if (name !== "") {
            sendInput({
                "key": name,
                "mods": mods
            });
            return;
        }
        if ((mods & 5) === 5)
            return;
        const text = (mods & 4) ? controlText(event.key) : event.text;
        if (text === "")
            return;
        sendInput({
            "text": text,
            "mods": mods
        });
    }

    Keys.onShortcutOverride: event => event.accepted = event.key !== Qt.Key_F12
    Keys.onPressed: event => {
        if (event.key === Qt.Key_F12)
            return;
        handleKey(event);
        event.accepted = true;
    }

    function isFixedWidth(probe) {
        return probe.font.family !== "" && Math.abs(probe.advanceWidth("M") - probe.advanceWidth("i")) < 0.01;
    }

    FontMetrics {
        id: configured

        font.family: term.behavior.fontFamily ?? ""
        font.pixelSize: Theme.fontSizeMedium
    }

    FontMetrics {
        id: themed

        font.family: Theme.monoFontFamily
        font.pixelSize: Theme.fontSizeMedium
    }

    FontMetrics {
        id: metrics

        font.family: term.isFixedWidth(configured) ? configured.font.family : term.isFixedWidth(themed) ? themed.font.family : "monospace"
        font.pixelSize: term.baseFontSize + SessionData.terminalZoom
    }

    TextMetrics {
        id: glyph

        font: metrics.font
        text: "M"
    }

    Timer {
        id: resizeDebounce

        interval: FilesMetrics.terminalResizeDelay
        onTriggered: term.redraw()
    }

    Connections {
        target: DuxFilesService

        function onConnectedChanged() {
            if (DuxFilesService.connected || term.sessionId === "")
                return;
            term.sessionId = "";
            term.exited();
        }
    }

    TapHandler {
        onPressedChanged: {
            if (pressed)
                term.forceActiveFocus();
        }
    }

    WheelHandler {
        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        onWheel: event => {
            if (term.mouse > 0) {
                term.wheelRemainder += event.angleDelta.y / 120;
                const notches = Math.trunc(term.wheelRemainder);
                term.wheelRemainder -= notches;
                const cell = term.cellAt(screen.mapFromItem(term, event.point.position));
                for (let i = 0; i < Math.min(Math.abs(notches), FilesMetrics.terminalWheelLines * 3); i++)
                    term.reportMouse("wheel", cell, notches > 0 ? "wheelUp" : "wheelDown", event.modifiers);
                return;
            }
            term.wheelRemainder += event.angleDelta.y / 120 * FilesMetrics.terminalWheelLines;
            const lines = Math.trunc(term.wheelRemainder);
            term.wheelRemainder -= lines;
            if (lines !== 0)
                term.request("terminal.scroll", {
                    "lines": lines
                });
        }
    }

    component UnderlineMark: Item {
        id: mark

        required property int style
        required property color tint
        readonly property real thickness: Math.max(1, Math.round(term.cellHeight / 16))
        readonly property real amplitude: thickness
        readonly property real lineY: height - thickness * 2 - (style === 3 ? amplitude : 0)

        function wave() {
            const points = [];
            const steps = Math.max(2, Math.ceil(width / 2));
            for (let i = 0; i <= steps; i++)
                points.push(Qt.point(i * width / steps, lineY + amplitude * Math.sin(2 * Math.PI * i / steps * Math.max(1, Math.round(width / term.cellWidth)))));
            return points;
        }

        Rectangle {
            visible: mark.style === 1 || mark.style === 2
            width: mark.width
            height: mark.thickness
            y: mark.lineY
            color: mark.tint
        }

        Rectangle {
            visible: mark.style === 2
            width: mark.width
            height: mark.thickness
            y: mark.lineY - mark.thickness * 2
            color: mark.tint
        }

        Repeater {
            model: mark.style === 4 || mark.style === 5 ? Math.ceil(mark.width / (mark.thickness * (mark.style === 4 ? 2 : 5))) : 0

            Rectangle {
                required property int index

                x: index * mark.thickness * (mark.style === 4 ? 2 : 5)
                y: mark.lineY
                width: mark.style === 4 ? mark.thickness : mark.thickness * 3
                height: mark.thickness
                color: mark.tint
            }
        }

        Shape {
            visible: mark.style === 3
            anchors.fill: parent

            ShapePath {
                strokeColor: mark.tint
                strokeWidth: mark.thickness
                fillColor: "transparent"

                PathPolyline {
                    path: mark.style === 3 ? mark.wave() : []
                }
            }
        }
    }

    component PlacedImage: Image {
        required property var modelData

        x: modelData.x * term.cellWidth + (modelData.ox ?? 0)
        y: modelData.y * term.cellHeight + (modelData.oy ?? 0)
        width: modelData.w * term.cellWidth
        height: modelData.h * term.cellHeight
        source: term.images[modelData.i] ?? ""
        sourceClipRect: Qt.rect(modelData.sx, modelData.sy, modelData.sw, modelData.sh)
        fillMode: Image.Stretch
        asynchronous: true
    }

    Item {
        id: screen

        anchors.fill: parent
        anchors.margins: FilesMetrics.terminalPadding

        Repeater {
            model: term.placements.filter(p => (p.z ?? 0) < 0)

            PlacedImage {}
        }

        Repeater {
            model: term.rows

            Item {
                id: line

                required property int index
                property var runs: term.lineStore[index] ?? []

                y: index * term.cellHeight
                width: screen.width
                height: term.cellHeight

                Connections {
                    target: term

                    function onLineUpdated(y, runs) {
                        if (y === line.index)
                            line.runs = runs;
                    }
                }

                Repeater {
                    model: line.runs

                    Item {
                        id: cell

                        required property var modelData
                        readonly property int attrs: modelData.a ?? 0
                        readonly property color ink: modelData.f || term.ink
                        readonly property color paper: modelData.b || term.paper
                        readonly property bool inverse: (attrs & 32) !== 0
                        readonly property int underline: modelData.u ?? 0
                        readonly property bool plainUnderline: underline === 1 && !modelData.uc

                        x: modelData.x * term.cellWidth
                        width: modelData.w * term.cellWidth
                        height: line.height

                        Rectangle {
                            anchors.fill: parent
                            visible: cell.inverse || !!cell.modelData.b
                            color: cell.inverse ? cell.ink : cell.paper
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            text: cell.modelData.t
                            textFormat: Text.PlainText
                            color: cell.inverse ? cell.paper : cell.ink
                            opacity: (cell.attrs & 2) ? Theme.pendingOpacity : 1
                            font.family: metrics.font.family
                            font.pixelSize: metrics.font.pixelSize
                            font.bold: (cell.attrs & 1) !== 0
                            font.italic: (cell.attrs & 4) !== 0
                            font.underline: cell.plainUnderline
                            font.strikeout: (cell.attrs & 16) !== 0
                        }

                        Loader {
                            anchors.fill: parent
                            active: cell.underline > 0 && !cell.plainUnderline
                            sourceComponent: UnderlineMark {
                                style: cell.underline
                                tint: cell.modelData.uc || (cell.inverse ? cell.paper : cell.ink)
                            }
                        }
                    }
                }
            }
        }

        Repeater {
            model: term.placements.filter(p => (p.z ?? 0) >= 0)

            PlacedImage {}
        }

        Repeater {
            model: term.selStart === null ? 0 : term.selEnd.y - term.selStart.y + 1

            Rectangle {
                id: highlight

                required property int index
                readonly property int row: (term.selStart?.y ?? 0) + index
                readonly property int first: index === 0 ? (term.selStart?.x ?? 0) : 0
                readonly property int last: row === term.selEnd?.y ? term.selEnd.x : term.cols - 1

                x: first * term.cellWidth
                y: row * term.cellHeight
                width: (last - first + 1) * term.cellWidth
                height: term.cellHeight
                color: term.customSelection ? term.selectionPaper : Theme.primary
                opacity: term.customSelection ? 1 : Theme.pendingOpacity
                clip: true

                Repeater {
                    model: term.customSelection ? (term.lineStore[highlight.row] ?? []) : []

                    Text {
                        required property var modelData

                        anchors.verticalCenter: parent.verticalCenter
                        x: modelData.x * term.cellWidth - highlight.x
                        text: modelData.t
                        textFormat: Text.PlainText
                        color: term.colors.selectionFg ? term.colors.selectionFg : term.colors.selectionBg ? (modelData.f || term.ink) : term.paper
                        font.family: metrics.font.family
                        font.pixelSize: metrics.font.pixelSize
                        font.bold: ((modelData.a ?? 0) & 1) !== 0
                        font.italic: ((modelData.a ?? 0) & 4) !== 0
                    }
                }
            }
        }

        MouseArea {
            id: pointer

            property int lastCell: -1
            property bool reporting: false

            anchors.fill: parent
            hoverEnabled: true
            cursorShape: term.linkHover ? Qt.PointingHandCursor : term.mouse > 0 ? Qt.ArrowCursor : Qt.IBeamCursor
            acceptedButtons: Qt.LeftButton | Qt.MiddleButton | Qt.RightButton
            onPressed: event => {
                term.forceActiveFocus();
                const cell = term.cellAt(Qt.point(event.x, event.y));
                lastCell = cell.y * term.cols + cell.x;
                reporting = term.mouse > 0 && !(event.modifiers & Qt.ShiftModifier);
                const link = (event.modifiers & Qt.ControlModifier) && event.button === Qt.LeftButton && !reporting ? term.linkAt(cell) : "";
                if (link !== "") {
                    term.openLink(link);
                    reporting = false;
                    return;
                }
                if (reporting) {
                    term.clearSelection();
                    term.reportMouse("press", cell, term.buttonName(event.button), event.modifiers);
                } else if (event.button === Qt.LeftButton)
                    term.pressSelection(cell);
            }
            onReleased: event => {
                if (reporting)
                    term.reportMouse("release", term.cellAt(Qt.point(event.x, event.y)), term.buttonName(event.button), event.modifiers);
                else if (event.button === Qt.LeftButton && term.behavior.copyOnSelect)
                    term.copySelection();
            }
            onExited: term.linkHover = false
            onPositionChanged: event => {
                const cell = term.cellAt(Qt.point(event.x, event.y));
                const index = cell.y * term.cols + cell.x;
                term.linkHover = (event.modifiers & Qt.ControlModifier) !== 0 && !(term.mouse > 0 && !(event.modifiers & Qt.ShiftModifier)) && term.linkAt(cell) !== "";
                if (index === lastCell)
                    return;
                if (pressedButtons === 0) {
                    if (term.mouse < 3 || (event.modifiers & Qt.ShiftModifier))
                        return;
                    lastCell = index;
                    term.reportMouse("move", cell, "none", event.modifiers);
                    return;
                }
                lastCell = index;
                if (!reporting) {
                    if (pressedButtons & Qt.LeftButton)
                        term.dragSelection(cell);
                    return;
                }
                if (term.mouse >= 2)
                    term.reportMouse("move", cell, term.heldButton(pressedButtons), event.modifiers);
            }
        }

        Rectangle {
            readonly property bool hollow: !term.activeFocus
            readonly property string shape: hollow ? "block" : term.cursorShape

            visible: term.cursor.visible && term.sessionId !== "" && term.blinkVisible
            x: term.cursor.x * term.cellWidth
            y: term.cursor.y * term.cellHeight + (shape === "underline" ? term.cellHeight - height : 0)
            width: shape === "bar" ? Theme.outlineWidthFocused : term.cellWidth
            height: shape === "underline" ? Theme.outlineWidthFocused : term.cellHeight
            color: hollow ? "transparent" : term.cursorColor
            opacity: shape === "block" ? Theme.pendingOpacity : 1
            border.width: shape === "block" ? Theme.outlineWidthFocused : 0
            border.color: term.cursorColor
        }

        Timer {
            id: blinkTimer

            interval: FilesMetrics.terminalBlinkInterval
            repeat: true
            running: term.cursorBlinks && term.activeFocus && term.cursor.visible
            onTriggered: term.blinkVisible = !term.blinkVisible
            onRunningChanged: term.blinkVisible = true
        }
    }
}
