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
        if (data.rows !== rows || data.cols !== cols) {
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
        if (painted)
            return;
        painted = lines.length === rows;
        if (!painted)
            redraw();
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

    function handleKey(event) {
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
        id: metrics

        font.family: Theme.monoFontFamily
        font.pixelSize: Theme.fontSizeMedium
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
