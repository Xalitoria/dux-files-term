pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Window
import Quickshell
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

FloatingWindow {
    id: window

    signal hideRequested

    property var tabs: []
    property int currentTab: 0
    property int activePaneIndex: 0
    property bool splitView: false
    property real splitRatio: 0.5
    property bool showHidden: false
    property string sortKey: "name"
    property bool sortDescending: false
    property bool sidebarOverlayOpen: false
    property string pendingTrust: ""
    property string openRequest: ""
    property bool applyingTab: false

    readonly property string opScope: "main"
    readonly property var secondPane: splitLoader.item
    readonly property var activePane: splitView && activePaneIndex === 1 && secondPane ? secondPane : primaryPane
    readonly property string currentPath: activePane.path
    readonly property var currentEntry: activePane.currentEntry
    readonly property bool compact: width < FilesMetrics.sidebarOverlayBreakpoint
    readonly property bool sidebarDocked: !SettingsData.sidebarCollapsed && !compact
    readonly property bool dialogOpen: conflictDialog.visible || errorDialog.visible || confirmDialog.visible || renameDialog.visible || newFolderDialog.visible || openWithDialog.visible || executableDialog.visible || trustDialog.visible || unlockDialog.visible || connectDialog.visible || credentialDialog.visible
    readonly property bool shortcutsEnabled: !dialogOpen && !primaryPane.editingPath && !(secondPane?.editingPath ?? false)

    title: I18n.tr("Dank Files", "main window title")
    implicitWidth: SessionData.windowWidth
    implicitHeight: SessionData.windowHeight
    minimumSize: Qt.size(480, 320)
    color: Theme.surface
    visible: true

    onClosed: requestClose()
    onMaximizedChanged: SessionData.maximized = maximized

    onWidthChanged: {
        if (!maximized)
            SessionData.windowWidth = width;
    }

    onHeightChanged: {
        if (!maximized)
            SessionData.windowHeight = height;
    }

    onOpenRequestChanged: {
        if (openRequest === "")
            return;
        openPath(openRequest, SettingsData.openLocationIn === "tab");
        openRequest = "";
    }

    onCompactChanged: {
        if (!compact)
            sidebarOverlayOpen = false;
    }

    Component.onCompleted: {
        restoreSession();
        maximized = SessionData.maximized;
    }

    function requestClose() {
        if (SettingsData.closeBehavior === "quit") {
            DankFilesService.quit();
            return;
        }
        hideRequested();
    }

    function toggleSidebar() {
        if (compact) {
            sidebarOverlayOpen = !sidebarOverlayOpen;
            return;
        }
        SettingsData.sidebarCollapsed = !SettingsData.sidebarCollapsed;
    }

    function setSplitRatio(value) {
        splitRatio = Math.max(FilesMetrics.minimumPaneFraction, Math.min(1 - FilesMetrics.minimumPaneFraction, value));
        SessionData.splitRatio = splitRatio;
    }

    function resizeSplit(fraction) {
        setSplitRatio(splitRatio + fraction);
    }

    function tabState(index) {
        return tabs[index] ?? null;
    }

    function paneStates() {
        const stored = tabState(currentTab)?.panes ?? [];
        return [primaryPane.saveState(), secondPane ? secondPane.saveState() : (stored[1] ?? null)];
    }

    function captureTabs() {
        if (applyingTab || tabs.length === 0)
            return;
        const next = tabs.slice();
        next[currentTab] = {
            "panes": paneStates(),
            "active": activePaneIndex
        };
        tabs = next;
        SessionData.tabs = tabs;
        SessionData.currentTab = currentTab;
    }

    function applyTab(index) {
        const state = tabState(index);
        if (!state)
            return;
        applyingTab = true;
        primaryPane.restoreState(state.panes?.[0] ?? {
            "path": Paths.strip(Paths.home)
        });
        if (secondPane)
            secondPane.restoreState(state.panes?.[1] ?? {
                "path": primaryPane.path
            });
        activePaneIndex = secondPane ? (state.active ?? 0) : 0;
        applyingTab = false;
    }

    function selectTab(index) {
        if (index === currentTab || index < 0 || index >= tabs.length)
            return;
        captureTabs();
        currentTab = index;
        applyTab(index);
        SessionData.currentTab = currentTab;
    }

    function newTab(path) {
        captureTabs();
        const next = tabs.slice();
        next.push({
            "panes": [
                {
                    "path": path === "" ? currentPath : path
                },
                null],
            "active": 0
        });
        tabs = next;
        currentTab = next.length - 1;
        applyTab(currentTab);
        SessionData.tabs = tabs;
        SessionData.currentTab = currentTab;
    }

    function closeTab(index) {
        if (tabs.length <= 1)
            return;
        const next = tabs.slice();
        next.splice(index, 1);
        const target = index < currentTab ? currentTab - 1 : Math.min(currentTab, next.length - 1);
        tabs = next;
        currentTab = target;
        applyTab(target);
        SessionData.tabs = tabs;
        SessionData.currentTab = currentTab;
    }

    function cycleTab(step) {
        if (tabs.length < 2)
            return;
        selectTab((currentTab + step + tabs.length) % tabs.length);
    }

    function tabLabel(state) {
        const path = state?.panes?.[0]?.path ?? "";
        if (path === "" || path === Paths.strip(Paths.home))
            return I18n.tr("Home", "sidebar entry for the user's home folder");
        if (path === "/")
            return "/";
        return path.substring(path.lastIndexOf("/") + 1);
    }

    function restoreSession() {
        splitView = SessionData.splitView;
        splitRatio = SessionData.splitRatio;
        const stored = SessionData.tabs;
        tabs = Array.isArray(stored) && stored.length > 0 ? stored : [
            {
                "panes": [
                    {
                        "path": Paths.strip(Paths.home)
                    },
                    null],
                "active": 0
            }
        ];
        currentTab = Math.max(0, Math.min(SessionData.currentTab, tabs.length - 1));
        applyTab(currentTab);
    }

    function openPath(path, inNewTab) {
        if (path === "")
            return;
        if (inNewTab) {
            newTab(path);
            return;
        }
        activePane.navigate(path);
    }

    function focusPane(index) {
        activePaneIndex = splitView && secondPane ? index : 0;
        activePane.focusBody();
    }

    function swapPaneFocus() {
        if (!splitView || !secondPane)
            return;
        focusPane(activePaneIndex === 0 ? 1 : 0);
    }

    function toggleSplit() {
        captureTabs();
        splitView = !splitView;
        SessionData.splitView = splitView;
        if (!splitView) {
            activePaneIndex = 0;
            return;
        }
        Qt.callLater(() => {
            if (!window.secondPane)
                return;
            const stored = window.tabState(window.currentTab)?.panes?.[1];
            window.secondPane.restoreState(stored ?? {
                "path": window.primaryPane.path
            });
        });
    }

    function cycleSort() {
        switch (sortKey) {
        case "name":
            sortKey = "mtime";
            return;
        case "mtime":
            sortKey = "size";
            return;
        case "size":
            sortKey = "type";
            return;
        default:
            sortKey = "name";
        }
    }

    function selection() {
        return currentEntry ? [currentEntry.path] : [];
    }

    function reportError(response) {
        if (response.error)
            ToastService.info(FileFormat.operationError(response.code, response.error));
    }

    function activate(entry) {
        if (!entry)
            return;
        if (entry.isDir) {
            activePane.navigate(entry.path);
            return;
        }
        const canRun = SandboxService.available("run");
        if (entry.untrusted && canRun) {
            trustDialog.subject = entry.name;
            trustDialog.pending = entry.path;
            trustDialog.open();
            return;
        }
        if (canRun && FileFormat.isRunnable(entry.mime, entry.isExecutable)) {
            executableDialog.ask(entry.path, entry.name);
            return;
        }
        AppsService.open([entry.path], response => window.reportError(response));
    }

    function openWithCurrent() {
        if (!currentEntry)
            return;
        openWithDialog.ask(currentEntry.mime, [currentEntry.path]);
    }

    function openContainingFolder() {
        if (!currentEntry)
            return;
        const parent = currentEntry.path.substring(0, currentEntry.path.lastIndexOf("/"));
        activePane.navigate(parent === "" ? "/" : parent);
    }

    function openTerminalHere() {
        AppsService.openTerminal(currentPath, response => window.reportError(response));
    }

    function showMenuFor(index, pointX, pointY) {
        openWithMenu.close();
        const point = activePane.mapToItem(surface, pointX, pointY);
        fileMenu.openAt(point.x, point.y);
    }

    function loadOpenWithMenu() {
        openWithMenu.apps = [];
        if (!currentEntry)
            return;
        AppsService.appsForMime(currentEntry.mime, currentEntry.path, response => {
            if (response.error)
                return;
            openWithMenu.apps = (response.apps || []).slice(0, 5);
        });
    }

    function trashSelection() {
        const paths = selection();
        if (paths.length === 0)
            return;
        FileOpsService.trash(paths, opScope);
    }

    function deleteSelection() {
        const paths = selection();
        if (paths.length === 0)
            return;
        confirmDialog.pending = paths;
        confirmDialog.subject = currentEntry.name;
        confirmDialog.open();
    }

    function renameSelection() {
        if (!currentEntry)
            return;
        renameDialog.target = currentEntry.path;
        renameDialog.ask(currentEntry.name);
    }

    function toastFor(job) {
        switch (job.kind) {
        case "trash":
            return I18n.tr("Moved to trash", "toast after trashing files");
        case "move":
            return I18n.tr("Moved", "toast after moving files");
        case "copy":
            return I18n.tr("Copied", "toast after copying files");
        case "rename":
            return I18n.tr("Renamed", "toast after renaming a file");
        default:
            return "";
        }
    }

    Connections {
        target: DevicesService

        function onRequestFailed(message, code) {
            ToastService.info(FileFormat.operationError(code, message));
        }
    }

    Connections {
        target: NetworkService

        function onRequestFailed(message, code) {
            ToastService.info(FileFormat.operationError(code, message));
        }

        function onAskRequested(ask) {
            credentialDialog.present(ask);
        }
    }

    Connections {
        target: FileOpsService

        function onRequestFailed(message) {
            ToastService.info(message);
        }

        function onJobFinished(job) {
            if (job.kind === "setPermissions" && window.pendingTrust !== "") {
                const target = window.pendingTrust;
                window.pendingTrust = "";
                if (job.state !== "failed")
                    AppsService.run(target, response => window.reportError(response));
            }
            if (job.state === "failed") {
                ToastService.info(FileFormat.operationError(job.code, job.error));
                return;
            }
            DevicesService.refreshUsage(true);
            if (job.cancelled || job.scope !== window.opScope || !job.undoable)
                return;
            const message = window.toastFor(job);
            if (message === "")
                return;
            ToastService.show(message, {
                "actionLabel": I18n.tr("Undo", "toast action reversing the last operation"),
                "action": () => FileOpsService.undo(window.opScope)
            });
        }
    }

    FloatingWindowControls {
        id: windowControls

        targetWindow: window
    }

    Component {
        id: secondPaneComponent

        FilesPane {
            active: window.activePaneIndex === 1
            showHidden: window.showHidden
            sortKey: window.sortKey
            sortDescending: window.sortDescending
            showActiveOutline: window.splitView
            onActivateRequested: entry => window.activate(entry)
            onContextMenuRequested: (index, pointX, pointY) => window.showMenuFor(index, pointX, pointY)
            onFocusRequested: window.activePaneIndex = 1
            onPathChanged: window.captureTabs()
        }
    }

    Item {
        id: surface

        anchors.fill: parent
        readonly property bool isFloatingWindowSurface: true
        readonly property bool windowActive: Window.active

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        onWindowActiveChanged: {
            if (!windowActive)
                return;
            primaryPane.refresh();
            window.secondPane?.refresh();
        }

        DankWindowHeader {
            id: header

            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            controls: windowControls
            title: window.title
            onCloseRequested: window.requestClose()

            DankIconButton {
                id: jobsButton

                anchors.verticalCenter: parent.verticalCenter
                visible: FileOpsService.jobs.length > 0
                iconName: FileOpsService.activeCount > 0 ? "sync" : "check_circle"
                size: "s"
                onClicked: jobsPopover.visible ? jobsPopover.close() : jobsPopover.open()

                JobsPopover {
                    id: jobsPopover

                    y: jobsButton.height + Theme.spacingS
                    x: jobsButton.width - width
                }
            }
        }

        Item {
            id: sidebarSlot

            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            width: window.sidebarDocked ? FilesMetrics.sidebarWidth : 0
            clip: true

            Behavior on width {
                enabled: Theme.animationsEnabled
                NumberAnimation {
                    duration: Theme.expressiveDurations.expressiveDefaultSpatial
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Theme.expressiveCurves.expressiveDefaultSpatial
                }
            }

            Sidebar {
                id: dockedSidebar

                anchors.fill: parent
                anchors.margins: FilesMetrics.sidebarPadding
                visible: parent.width > 0
                currentPath: window.currentPath
                onNavigateRequested: path => window.openPath(path, false)
                onUnlockRequested: device => unlockDialog.ask(device)
                onConnectRequested: connectDialog.ask()
            }
        }

        Rectangle {
            anchors.left: sidebarSlot.right
            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            width: Theme.dividerWidth
            visible: sidebarSlot.width > 0
            color: Theme.outlineVariant
        }

        Item {
            id: content

            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            anchors.left: sidebarSlot.right
            anchors.right: parent.right
            anchors.margins: FilesMetrics.contentMargin

            TabRow {
                id: tabRow

                anchors.top: parent.top
                anchors.left: parent.left
                anchors.right: parent.right
                visible: window.tabs.length > 1
                height: visible ? implicitHeight : 0
                tabs: window.tabs.map(state => window.tabLabel(state))
                currentIndex: window.currentTab
                onTabSelected: index => window.selectTab(index)
                onTabClosed: index => window.closeTab(index)
                onTabAdded: window.newTab("")
            }

            FilesTopbar {
                id: topbar

                anchors.top: tabRow.bottom
                anchors.topMargin: tabRow.visible ? FilesMetrics.contentSpacing : 0
                anchors.left: parent.left
                anchors.right: parent.right
                splitView: window.splitView
                bookmarked: BookmarksService.has(window.currentPath)
                backEnabled: window.activePane.canGoBack
                forwardEnabled: window.activePane.canGoForward
                sortKey: window.sortKey
                sortDescending: window.sortDescending
                onSidebarToggled: window.toggleSidebar()
                onBackRequested: window.activePane.back()
                onForwardRequested: window.activePane.forward()
                onSplitToggled: window.toggleSplit()
                onBookmarkToggled: BookmarksService.toggle(window.currentPath)
                onSortCycled: window.cycleSort()
                onSortDirectionToggled: window.sortDescending = !window.sortDescending
                onCopyPathRequested: Paths.copyPathToClipboard(window.currentPath)
                onReloadRequested: window.activePane.refresh()
                onTerminalRequested: window.openTerminalHere()
                onPropertiesRequested: ToastService.info(I18n.tr("Not available", "status for a feature the environment does not allow"))
            }

            Item {
                id: panesRow

                anchors.top: topbar.bottom
                anchors.topMargin: FilesMetrics.contentSpacing
                anchors.bottom: parent.bottom
                anchors.left: parent.left
                anchors.right: parent.right

                readonly property real usableWidth: width - (window.splitView ? FilesMetrics.splitterWidth : 0)

                FilesPane {
                    id: primaryPane

                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.left: parent.left
                    width: window.splitView ? panesRow.usableWidth * window.splitRatio : panesRow.width
                    active: window.activePaneIndex === 0
                    showHidden: window.showHidden
                    sortKey: window.sortKey
                    sortDescending: window.sortDescending
                    showActiveOutline: window.splitView
                    onActivateRequested: entry => window.activate(entry)
                    onContextMenuRequested: (index, pointX, pointY) => window.showMenuFor(index, pointX, pointY)
                    onFocusRequested: window.activePaneIndex = 0
                    onPathChanged: window.captureTabs()
                }

                PaneSplitter {
                    id: splitter

                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.left: primaryPane.right
                    visible: window.splitView
                    onMoved: delta => window.resizeSplit(delta / Math.max(1, panesRow.usableWidth))
                    onReset: window.setSplitRatio(0.5)
                }

                Loader {
                    id: splitLoader

                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.left: splitter.right
                    anchors.right: parent.right
                    active: window.splitView
                    sourceComponent: secondPaneComponent
                }
            }
        }

        Rectangle {
            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            visible: opacity > 0
            opacity: window.sidebarOverlayOpen ? 1 : 0
            color: Theme.withAlpha(Theme.scrimColor, Theme.scrimAlpha)

            Behavior on opacity {
                enabled: Theme.animationsEnabled
                NumberAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            MouseArea {
                anchors.fill: parent
                onClicked: window.sidebarOverlayOpen = false
            }
        }

        Rectangle {
            id: sidebarOverlay

            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            width: FilesMetrics.sidebarWidth
            visible: x > -width
            x: window.sidebarOverlayOpen ? 0 : -width
            color: Theme.surfaceContainer

            Behavior on x {
                enabled: Theme.animationsEnabled
                NumberAnimation {
                    duration: Theme.expressiveDurations.expressiveDefaultSpatial
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Theme.expressiveCurves.expressiveDefaultSpatial
                }
            }

            Sidebar {
                anchors.fill: parent
                anchors.margins: FilesMetrics.sidebarPadding
                currentPath: window.currentPath
                onNavigateRequested: path => {
                    window.sidebarOverlayOpen = false;
                    window.openPath(path, false);
                }
                onUnlockRequested: device => unlockDialog.ask(device)
                onConnectRequested: connectDialog.ask()
            }
        }
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Return", "Enter"]
        onActivated: window.activate(window.currentEntry)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+Return", "Ctrl+Enter"]
        onActivated: window.openContainingFolder()
    }

    Shortcut {
        enabled: window.shortcutsEnabled && window.currentEntry !== null
        sequences: ["Shift+F10", "Menu"]
        onActivated: window.showMenuFor(window.activePane.currentIndex, Theme.spacingL, Theme.spacingL)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Alt+Left"]
        onActivated: window.activePane.back()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Alt+Right"]
        onActivated: window.activePane.forward()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Alt+Up"]
        onActivated: window.activePane.up()
    }

    Shortcut {
        enabled: !window.dialogOpen
        sequences: ["Ctrl+L"]
        onActivated: window.activePane.editPath()
    }

    Shortcut {
        enabled: !window.dialogOpen
        sequences: ["Ctrl+F"]
        onActivated: topbar.focusSearch()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+T"]
        onActivated: window.newTab("")
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+W"]
        onActivated: window.closeTab(window.currentTab)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+Tab"]
        onActivated: window.cycleTab(1)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+Shift+Tab"]
        onActivated: window.cycleTab(-1)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["F5"]
        onActivated: window.activePane.refresh()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["F6"]
        onActivated: window.swapPaneFocus()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+Shift+O"]
        onActivated: window.openWithCurrent()
    }

    Shortcut {
        enabled: window.shortcutsEnabled && SandboxService.available("terminal")
        sequences: ["Ctrl+Alt+T"]
        onActivated: window.openTerminalHere()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+H"]
        onActivated: window.showHidden = !window.showHidden
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+C"]
        onActivated: FileOpsService.copyToClipboard(window.selection())
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+X"]
        onActivated: FileOpsService.cut(window.selection())
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+V"]
        onActivated: FileOpsService.paste(window.currentPath, window.opScope)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["F2"]
        onActivated: window.renameSelection()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Delete"]
        onActivated: window.trashSelection()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Shift+Delete"]
        onActivated: window.deleteSelection()
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+Z"]
        onActivated: FileOpsService.undo(window.opScope)
    }

    Shortcut {
        enabled: window.shortcutsEnabled
        sequences: ["Ctrl+Shift+N"]
        onActivated: newFolderDialog.ask(I18n.tr("Untitled folder", "default name for a new folder"))
    }

    ContextMenu {
        id: fileMenu

        anchorItem: surface

        ContextMenuItem {
            text: I18n.tr("Open", "context menu entry opening a file")
            iconName: "open_in_new"
            onClicked: {
                fileMenu.close();
                window.activate(window.currentEntry);
            }
        }

        ContextMenuItem {
            id: openWithItem

            text: I18n.tr("Open with", "open with dialog header")
            iconName: "apps"
            submenu: true
            enabled: window.currentEntry !== null && !window.currentEntry.isDir
            onEntered: {
                if (!enabled)
                    return;
                window.loadOpenWithMenu();
                openWithMenu.openAt(fileMenu.x + fileMenu.width - Theme.spacingM, fileMenu.y + y);
            }
            onClicked: entered()
        }

        ContextMenuItem {
            text: I18n.tr("Open in new tab", "context menu entry opening a folder in a new tab")
            iconName: "tab"
            enabled: window.currentEntry !== null && window.currentEntry.isDir
            onClicked: {
                fileMenu.close();
                window.newTab(window.currentEntry.path);
            }
        }

        ContextMenuItem {
            visible: SandboxService.available("terminal")
            text: I18n.tr("Open in terminal", "context menu entry opening a terminal")
            iconName: "terminal"
            onClicked: {
                fileMenu.close();
                window.openTerminalHere();
            }
        }
    }

    ContextMenu {
        id: openWithMenu

        property var apps: []

        anchorItem: surface

        Repeater {
            model: openWithMenu.apps

            ContextMenuItem {
                required property var modelData

                text: modelData.name
                iconName: modelData.isDefault ? "check" : ""
                onClicked: {
                    openWithMenu.close();
                    fileMenu.close();
                    AppsService.openWith(window.selection(), modelData.id, response => window.reportError(response));
                }
            }
        }

        ContextMenuItem {
            text: I18n.tr("Other application…", "open with menu entry opening the full application picker")
            iconName: "more_horiz"
            onClicked: {
                openWithMenu.close();
                fileMenu.close();
                window.openWithCurrent();
            }
        }
    }

    OpenWithDialog {
        id: openWithDialog
    }

    ExecutableDialog {
        id: executableDialog

        onRunRequested: AppsService.run(path, response => window.reportError(response))
        onEditRequested: openWithDialog.ask("text/plain", [path])
    }

    ConfirmDialog {
        id: trustDialog

        property string pending: ""

        title: I18n.tr("Run this desktop file?", "untrusted desktop file dialog header")
        message: I18n.tr("%1 has not been marked as trusted. Running it allows it to start any program.", "untrusted desktop file dialog explanation").arg(subject)
        confirmText: I18n.tr("Trust and run", "untrusted desktop file dialog action")
        danger: false
        onConfirmed: {
            window.pendingTrust = pending;
            FileOpsService.setPermissions([pending], "755", false);
            pending = "";
        }
    }

    ConflictDialog {
        id: conflictDialog
    }

    OperationErrorDialog {
        id: errorDialog
    }

    NameDialog {
        id: renameDialog

        property string target: ""

        title: I18n.tr("Rename", "rename dialog header")
        acceptText: I18n.tr("Rename", "rename dialog header")
        onSubmitted: name => FileOpsService.rename(target, name, window.opScope)
    }

    NameDialog {
        id: newFolderDialog

        title: I18n.tr("New folder", "new folder dialog header")
        acceptText: I18n.tr("Create", "dialog button creating an item")
        onSubmitted: name => FileOpsService.newFolder(window.currentPath, name, window.opScope)
    }

    ConfirmDialog {
        id: confirmDialog

        property var pending: []

        title: I18n.tr("Delete permanently?", "permanent delete confirmation header")
        message: I18n.tr("%1 will be deleted permanently. This cannot be undone.", "permanent delete confirmation explanation").arg(subject)
        onConfirmed: {
            FileOpsService.deletePermanently(pending);
            pending = [];
        }
    }

    UnlockDialog {
        id: unlockDialog

        onUnlocked: mountPoint => {
            if (mountPoint !== "")
                window.openPath(mountPoint, false);
        }
    }

    ConnectServerDialog {
        id: connectDialog

        onConnected: path => {
            if (path !== "")
                window.openPath(path, false);
        }
    }

    CredentialDialog {
        id: credentialDialog
    }

    Toast {
        anchors.fill: surface
    }
}
