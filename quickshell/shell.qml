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

    property string pendingPath: ""

    Component.onCompleted: {
        Quickshell.watchFiles = enableHotReload;
        DC.Style.theme = Theme;
        DC.Style.settings = SettingsData;
        DC.I18n.backend = I18n;
        DC.Paths.backend = Paths;
        DC.Log.backend = Log;
        DC.Host.session = SessionService;
        DC.Host.cache = CacheData;
        DC.Host.files = DankFilesService;
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

    function handleOpenPath(path) {
        if (path === "")
            return;
        pendingPath = path;
        showAndFocus();
        applyPendingPath();
    }

    function applyPendingPath() {
        if (!windowLoader.item || pendingPath === "")
            return;
        windowLoader.item.currentPath = pendingPath;
        pendingPath = "";
    }

    Connections {
        target: DankFilesService
        function onWindowActionRequested(action) {
            root.handleWindowAction(action);
        }
        function onOpenPathRequested(path) {
            root.handleOpenPath(path);
        }
    }

    Connections {
        target: windowLoader
        function onItemChanged() {
            root.applyPendingPath();
        }
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
