pragma Singleton
pragma ComponentBehavior: Bound

import QtCore
import QtQuick
import Quickshell
import Quickshell.Io
import qs.Services

Singleton {
    id: root

    readonly property var log: Log.scoped("SessionData")

    readonly property url stateDir: `${StandardPaths.writableLocation(StandardPaths.GenericStateLocation)}/dfiles`
    readonly property string sessionPath: Paths.strip(stateDir) + "/session.json"

    property real windowWidth: 960
    property real windowHeight: 640
    property bool maximized: false
    property bool splitView: false
    property real splitRatio: 0.5
    property int currentTab: 0
    property var tabs: []
    property var recentDirs: []
    property int gridZoom: FilesMetrics.defaultGridZoom
    property int listZoom: FilesMetrics.defaultListZoom
    property var listColumns: ["size", "modified", "type"]
    property var columnWidths: ({})
    property var browsing: ({})

    readonly property var persistedState: ({
            "windowWidth": Math.round(windowWidth),
            "windowHeight": Math.round(windowHeight),
            "maximized": maximized,
            "splitView": splitView,
            "splitRatio": splitRatio,
            "currentTab": currentTab,
            "tabs": tabs,
            "recentDirs": recentDirs,
            "gridZoom": gridZoom,
            "listZoom": listZoom,
            "listColumns": listColumns,
            "columnWidths": columnWidths,
            "browsing": browsing
        })

    readonly property int recentLimit: 20

    property bool _loaded: false
    property bool _selfWrite: false

    function lastUsed(key, fallback) {
        return browsing?.[key] ?? fallback;
    }

    function remember(values) {
        const next = Object.assign({}, browsing, values);
        if (JSON.stringify(next) === JSON.stringify(browsing))
            return;
        browsing = next;
    }

    function recordRecent(path) {
        if (path === "" || recentDirs[0] === path)
            return;
        recentDirs = [path].concat(recentDirs.filter(entry => entry !== path)).slice(0, recentLimit);
    }

    onPersistedStateChanged: {
        if (!_loaded)
            return;
        writeDebounce.restart();
    }

    function _apply(text) {
        if (text === "") {
            _loaded = true;
            return;
        }

        let data = null;
        try {
            data = JSON.parse(text);
        } catch (e) {
            log.warn(`ignoring unreadable session state: ${e}`);
        }

        for (const key in persistedState) {
            if (!data || data[key] === undefined)
                continue;
            root[key] = data[key];
        }
        _loaded = true;
    }

    Component.onCompleted: {
        Paths.mkdir(stateDir);
        _apply(sessionFile.text() ?? "");
    }

    Timer {
        id: writeDebounce

        interval: 400
        onTriggered: {
            root._selfWrite = true;
            sessionFile.setText(JSON.stringify(root.persistedState, null, 1));
        }
    }

    FileView {
        id: sessionFile

        path: root.sessionPath
        blockLoading: true
        atomicWrites: true
        watchChanges: true
        printErrors: false

        onLoaded: root._apply(text())
        onFileChanged: {
            if (root._selfWrite) {
                root._selfWrite = false;
                return;
            }
            reload();
        }
    }
}
