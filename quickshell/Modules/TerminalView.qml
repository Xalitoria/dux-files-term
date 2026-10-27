pragma ComponentBehavior: Bound

import QtQuick
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
    property var cursor: ({
            "x": 0,
            "y": 0,
            "visible": false
        })
    property real wheelRemainder: 0
    readonly property var life: ({
            "alive": true
        })

    readonly property real cellWidth: Math.max(1, metrics.advanceWidth("M"))
    readonly property real cellHeight: Math.max(1, Math.ceil(metrics.height))
    readonly property int fitCols: Math.floor((width - 2 * FilesMetrics.terminalPadding) / cellWidth)
    readonly property int fitRows: Math.floor((height - 2 * FilesMetrics.terminalPadding) / cellHeight)

    signal exited
    signal failed(var response)
    signal lineUpdated(int y, var runs)

    radius: FilesMetrics.paneRadius
    color: Theme.foregroundColor(Theme.cardSurface, true)
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
    onFitColsChanged: resizeDebounce.restart()
    onFitRowsChanged: resizeDebounce.restart()

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
            "rows": fitRows
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
        cursor = data.cursor;
        mouse = data.mouse ?? 0;
        if (painted)
            return;
        painted = lines.length === rows;
        if (!painted)
            redraw();
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

    function zoomStep(key) {
        switch (key) {
        case Qt.Key_Plus:
        case Qt.Key_Equal:
            return 1;
        case Qt.Key_Minus:
        case Qt.Key_Underscore:
            return -1;
        }
        return 0;
    }

    function handleZoom(event) {
        if (modifierMask(event.modifiers) !== 5)
            return false;
        if (event.key === Qt.Key_Backspace) {
            setZoom(0);
            return true;
        }
        const step = zoomStep(event.key);
        if (step === 0)
            return false;
        setZoom(SessionData.terminalZoom + step);
        return true;
    }

    function handleKey(event) {
        if (handleZoom(event))
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
        if ((mods & 5) === 5) {
            if (event.key === Qt.Key_C)
                copySelection();
            if (event.key === Qt.Key_V)
                sendInput({
                    "text": Quickshell.clipboardText,
                    "paste": true
                });
            return;
        }
        const text = (mods & 4) ? controlText(event.key) : event.text;
        if (text === "")
            return;
        sendInput({
            "text": text,
            "mods": mods
        });
    }

    Keys.onShortcutOverride: event => event.accepted = event.key !== Qt.Key_F4
    Keys.onPressed: event => {
        if (event.key === Qt.Key_F4)
            return;
        handleKey(event);
        event.accepted = true;
    }

    FontMetrics {
        id: themed

        font.family: Theme.monoFontFamily
        font.pixelSize: Theme.fontSizeMedium
    }

    FontMetrics {
        id: metrics

        font.family: Math.abs(themed.advanceWidth("M") - themed.advanceWidth("i")) < 0.01 ? Theme.monoFontFamily : "monospace"
        font.pixelSize: Theme.fontSizeMedium + SessionData.terminalZoom
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

    Item {
        id: screen

        anchors.fill: parent
        anchors.margins: FilesMetrics.terminalPadding

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
                        readonly property color ink: modelData.f || Theme.surfaceText
                        readonly property color paper: modelData.b || term.color
                        readonly property bool inverse: (attrs & 32) !== 0

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
                            font.underline: (cell.attrs & 8) !== 0
                            font.strikeout: (cell.attrs & 16) !== 0
                        }
                    }
                }
            }
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
                color: Theme.primary
                opacity: Theme.pendingOpacity
            }
        }

        MouseArea {
            id: pointer

            property int lastCell: -1
            property bool reporting: false

            anchors.fill: parent
            hoverEnabled: term.mouse === 3
            cursorShape: term.mouse > 0 ? Qt.ArrowCursor : Qt.IBeamCursor
            acceptedButtons: Qt.LeftButton | Qt.MiddleButton | Qt.RightButton
            onPressed: event => {
                term.forceActiveFocus();
                const cell = term.cellAt(Qt.point(event.x, event.y));
                lastCell = cell.y * term.cols + cell.x;
                reporting = term.mouse > 0 && !(event.modifiers & Qt.ShiftModifier);
                if (reporting) {
                    term.clearSelection();
                    term.reportMouse("press", cell, term.buttonName(event.button), event.modifiers);
                } else if (event.button === Qt.LeftButton)
                    term.pressSelection(cell);
            }
            onReleased: event => {
                if (reporting)
                    term.reportMouse("release", term.cellAt(Qt.point(event.x, event.y)), term.buttonName(event.button), event.modifiers);
            }
            onPositionChanged: event => {
                const cell = term.cellAt(Qt.point(event.x, event.y));
                const index = cell.y * term.cols + cell.x;
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
            visible: term.cursor.visible && term.sessionId !== ""
            x: term.cursor.x * term.cellWidth
            y: term.cursor.y * term.cellHeight
            width: term.cellWidth
            height: term.cellHeight
            color: term.activeFocus ? Theme.primary : "transparent"
            opacity: Theme.pendingOpacity
            border.width: Theme.outlineWidthFocused
            border.color: Theme.primary
        }
    }
}
