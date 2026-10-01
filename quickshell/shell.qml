//@ pragma Env QSG_RENDER_LOOP=threaded
//@ pragma Env QT_WAYLAND_DISABLE_WINDOWDECORATION=1
//@ pragma Env QT_QUICK_CONTROLS_STYLE=Material
//@ pragma UseQApplication
//@ pragma AppId com.danklinux.dankfiles

pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.Common
import qs.DankCommon.Common as DC
import qs.Modules
import qs.Services

ShellRoot {
    id: root

    readonly property var log: Log.scoped("Shell")
    readonly property bool enableHotReload: Quickshell.env("DFILES_ENABLE_HOTRELOAD") === "1" || Quickshell.env("DFILES_ENABLE_HOTRELOAD") === "true"

    readonly property bool dbusActivated: Quickshell.env("DFILES_DBUS_ACTIVATED") === "1"
    property var pendingRequests: []
    property bool windowShown: false

    Component.onCompleted: {
        Quickshell.watchFiles = enableHotReload;
        DC.Style.theme = Theme;
        DC.Style.settings = SettingsData;
        DC.I18n.backend = I18n;
        DC.Paths.backend = Paths;
        DC.Log.backend = Log;
        DC.Host.session = SessionService;
        DC.Host.cache = CacheData;
        DC.Host.files = DirectoryService;
    }

    function ownToplevel() {
        const toplevels = ToplevelManager.toplevels;
        if (!toplevels || !toplevels.values)
            return null;
        for (const t of toplevels.values) {
            if (t.appId === "com.danklinux.dankfiles")
                return t;
        }
        return null;
    }

    function focusToplevel() {
        const t = ownToplevel();
        if (!t)
            return;
        if (t.minimized)
            t.minimized = false;
        t.activate();
    }

    function showAndFocus() {
        if (windowLoader.active) {
            focusToplevel();
            return;
        }
        windowLoader.active = true;
        focusRetry.restart();
    }

    function handleWindowAction(action) {
        switch (action) {
        case "show":
            showAndFocus();
            break;
        case "hide":
            windowLoader.active = false;
            break;
        case "toggle":
            {
                const t = windowLoader.active ? ownToplevel() : null;
                if (t && !t.minimized) {
                    windowLoader.active = false;
                    return;
                }
                showAndFocus();
                break;
            }
        }
    }

    function queueRequest(request) {
        pendingRequests = pendingRequests.concat([request]);
        showAndFocus();
        applyPendingRequests();
    }

    function applyPendingRequests() {
        const mainWindow = windowLoader.item as MainWindow;
        if (!mainWindow || pendingRequests.length === 0)
            return;
        const requests = pendingRequests;
        pendingRequests = [];
        for (const request of requests) {
            switch (request.kind) {
            case "open":
                mainWindow.openLocations(request.locations);
                break;
            case "properties":
                mainWindow.showPropertiesFor(request.paths);
                break;
            }
        }
    }

    Connections {
        target: DankFilesService
        function onWindowActionRequested(action) {
            root.handleWindowAction(action);
        }
        function onOpenRequested(locations) {
            if (locations.length === 0)
                return;
            root.queueRequest({
                "kind": "open",
                "locations": locations
            });
        }
        function onPropertiesRequested(paths) {
            if (paths.length === 0)
                return;
            root.queueRequest({
                "kind": "properties",
                "paths": paths
            });
        }
    }

    Connections {
        target: windowLoader
        function onItemChanged() {
            if (windowLoader.item)
                root.windowShown = true;
            root.applyPendingRequests();
        }
    }

    Timer {
        interval: 30000
        running: root.dbusActivated && !root.windowShown
        onTriggered: DankFilesService.quit()
    }

    Timer {
        id: focusRetry
        interval: 150
        repeat: false
        onTriggered: root.focusToplevel()
    }

    LazyLoader {
        id: windowLoader
        active: Quickshell.env("DFILES_START_HIDDEN") !== "1"

        MainWindow {
            onHideRequested: windowLoader.active = false
        }
    }
}
