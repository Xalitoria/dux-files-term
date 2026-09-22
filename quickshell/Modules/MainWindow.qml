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
    readonly property var cutSet: {
        const set = ({});
        if (!ClipboardService.cut)
            return set;
        for (const path of ClipboardService.paths)
            set[path] = true;
        return set;
    }
    readonly property bool shiftHeld: (window.activeModifiers & Qt.ShiftModifier) !== 0
    property int activeModifiers: Qt.NoModifier
    readonly property var secondPane: splitLoader.item
    readonly property var activePane: splitView && activePaneIndex === 1 && secondPane ? secondPane : primaryPane
    readonly property string currentPath: activePane.path
    readonly property bool compact: width < FilesMetrics.sidebarOverlayBreakpoint
    readonly property bool sidebarDocked: !SettingsData.sidebarCollapsed && !compact
    readonly property bool dialogOpen: conflictDialog.visible || errorDialog.visible || confirmDialog.visible || renameDialog.visible || newFolderDialog.visible || openWithDialog.visible || executableDialog.visible || trustDialog.visible || unlockDialog.visible || connectDialog.visible || credentialDialog.visible || propertiesDialog.visible || shortcutsOverlay.visible
    readonly property bool shortcutsEnabled: !dialogOpen && !primaryPane.editingPath && !(secondPane?.editingPath ?? false) && !primaryPane.renaming && !(secondPane?.renaming ?? false)

    title: I18n.tr("Files", "main window title")
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

    function setSort(key) {
        if (key === "")
            return;
        if (sortKey === key) {
            sortDescending = !sortDescending;
            return;
        }
        sortKey = key;
        sortDescending = false;
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

    // The one place that decides what an action does. The keyboard table, the
    // context menus and the topbar all dispatch through it.
    function runAction(id) {
        switch (id) {
        case "open":
            activate(firstSelected());
            return;
        case "openContainingFolder":
            openContainingFolder();
            return;
        case "openWith":
            openWithCurrent();
            return;
        case "openTerminal":
            openTerminalHere();
            return;
        case "newTab":
            newTab(firstSelected()?.path ?? "");
            return;
        case "openInPane":
            openInOtherPane(firstSelected()?.path ?? "");
            return;
        case "closeTab":
            closeTab(currentTab);
            return;
        case "nextTab":
            cycleTab(1);
            return;
        case "previousTab":
            cycleTab(-1);
            return;
        case "back":
            activePane.back();
            return;
        case "forward":
            activePane.forward();
            return;
        case "editPath":
            activePane.editPath();
            return;
        case "search":
            topbar.focusSearch();
            return;
        case "swapPane":
            swapPaneFocus();
            return;
        case "listView":
            SessionData.viewMode = "list";
            return;
        case "gridView":
            SessionData.viewMode = "grid";
            return;
        case "zoomIn":
            activePane.body.zoomBy(1);
            return;
        case "zoomOut":
            activePane.body.zoomBy(-1);
            return;
        case "showHidden":
            showHidden = !showHidden;
            return;
        case "reload":
            activePane.refresh();
            return;
        case "shortcuts":
            shortcutsOverlay.open();
            return;
        case "selectAll":
            activePane.selection.selectAll(activePane.body.directory.entries);
            return;
        case "invertSelection":
            activePane.selection.invert(activePane.body.directory.entries);
            return;
        case "copy":
            ClipboardService.copy(selectedPaths());
            return;
        case "cut":
            ClipboardService.cutPaths(selectedPaths());
            return;
        case "paste":
            pasteInto(currentPath, false);
            return;
        case "pasteInto":
            pasteInto(firstSelected()?.path ?? currentPath, false);
            return;
        case "pasteLink":
            pasteInto(currentPath, true);
            return;
        case "rename":
            renameSelection();
            return;
        case "trash":
            trashSelection();
            return;
        case "deletePermanently":
            deleteSelection();
            return;
        case "undo":
            FileOpsService.undo(opScope);
            return;
        case "newFolder":
            newFolderDialog.ask(I18n.tr("Untitled folder", "default name for a new folder"));
            return;
        case "properties":
            showProperties();
            return;
        case "copyPath":
            Paths.copyPathToClipboard(selectedPaths().join("\n"));
            return;
        case "contextMenu":
            showItemMenu(Theme.spacingL, Theme.spacingL, Qt.NoModifier);
            return;
        }
    }

    function openInOtherPane(path) {
        if (path === "")
            return;
        if (!splitView)
            toggleSplit();
        Qt.callLater(() => {
            const target = window.activePaneIndex === 0 ? window.secondPane : window.primaryPane;
            target?.navigate(path);
        });
    }

    function selectedPaths() {
        return activePane.selection.paths;
    }

    function selectedEntries() {
        return activePane.selectedEntries;
    }

    function selectionIsDirectory() {
        const entries = selectedEntries();
        return entries.length > 0 && entries.every(entry => entry.isDir);
    }

    function selectionIsFile() {
        const entries = selectedEntries();
        return entries.length > 0 && entries.every(entry => !entry.isDir);
    }

    function firstSelected() {
        return selectedEntries()[0] ?? null;
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
        const entry = firstSelected();
        if (!entry)
            return;
        openWithDialog.ask(entry.mime, selectedPaths());
    }

    function openContainingFolder() {
        const entry = firstSelected();
        if (!entry)
            return;
        const parent = entry.path.substring(0, entry.path.lastIndexOf("/"));
        activePane.navigate(parent === "" ? "/" : parent);
    }

    function openTerminalHere() {
        AppsService.openTerminal(currentPath, response => window.reportError(response));
    }

    function showItemMenu(pointX, pointY, modifiers) {
        activeModifiers = modifiers ?? Qt.NoModifier;
        openWithMenu.close();
        backgroundMenu.close();
        const point = activePane.mapToItem(surface, pointX, pointY);
        itemMenu.openAt(point.x, point.y);
    }

    function showColumnsMenu(pointX, pointY) {
        itemMenu.close();
        backgroundMenu.close();
        const point = activePane.mapToItem(surface, pointX, pointY);
        columnsMenu.openAt(point.x, point.y);
    }

    function toggleColumn(id) {
        const shown = SessionData.listColumns;
        SessionData.listColumns = shown.includes(id) ? shown.filter(entry => entry !== id) : shown.concat([id]);
    }

    function showBackgroundMenu(pointX, pointY) {
        openWithMenu.close();
        itemMenu.close();
        const point = activePane.mapToItem(surface, pointX, pointY);
        backgroundMenu.openAt(point.x, point.y);
    }

    function loadOpenWithMenu() {
        openWithMenu.apps = [];
        const entry = firstSelected();
        if (!entry)
            return;
        AppsService.appsForMime(entry.mime, entry.path, response => {
            if (response.error)
                return;
            openWithMenu.apps = (response.apps || []).slice(0, 5);
        });
    }

    function trashSelection() {
        const paths = selectedPaths();
        if (paths.length === 0)
            return;
        FileOpsService.trash(paths, opScope);
    }

    function deleteSelection() {
        const paths = selectedPaths();
        if (paths.length === 0)
            return;
        confirmDialog.pending = paths;
        confirmDialog.subject = paths.length === 1 ? firstSelected().name : FileFormat.count(paths.length);
        confirmDialog.open();
    }

    function renameSelection() {
        const entry = firstSelected();
        if (!entry)
            return;
        activePane.beginRename(entry.path);
    }

    function pasteInto(dest, asLink) {
        ClipboardService.read(result => {
            if (result.paths.length === 0)
                return;
            if (asLink) {
                FileOpsService.link(result.paths, dest, window.opScope);
                return;
            }
            if (result.cut) {
                FileOpsService.move(result.paths, dest, window.opScope);
                ClipboardService.clear();
                return;
            }
            FileOpsService.copy(result.paths, dest, window.opScope);
        });
    }

    function showProperties() {
        const entries = selectedEntries();
        if (entries.length > 0) {
            propertiesDialog.show(entries);
            return;
        }
        DirectoryService.list(currentPath, {
            "limit": 1
        }, () => {});
        DankFilesService.sendRequest("files.stat", {
            "path": currentPath
        }, response => {
            if (response.error) {
                ToastService.info(FileFormat.operationError(response.code, response.error));
                return;
            }
            propertiesDialog.show([(response.result || {}).entry]);
        });
    }

    function applyDrop(paths, dest, action) {
        switch (action) {
        case "copy":
            FileOpsService.copy(paths, dest, opScope);
            return;
        case "move":
            FileOpsService.move(paths, dest, opScope);
            return;
        case "ask":
            dropMenu.present(paths, dest);
            return;
        }
        FileOpsService.transport.sendRequest("files.op.dropAction", {
            "paths": paths,
            "dest": dest
        }, response => {
            const resolved = (response.result || {}).action === "move" ? "move" : "copy";
            window.applyDrop(paths, dest, resolved);
        });
    }

    function beginDrag(paths) {
        if (paths.length === 0)
            return;
        const entry = window.firstSelected();
        DragService.begin(paths, currentPath);
        dragGhost.count = paths.length;
        dragGhost.iconName = entry?.iconName ?? "";
        dragGhost.thumbnail = entry?.thumbnail ?? "";
        dragGhost.mime = entry?.mime ?? "";
        dragGhost.path = entry?.path ?? "";
        dragGhost.isDir = entry?.isDir ?? false;
        dragSource.Drag.mimeData = DragService.mimeData();
        dragGhost.grabToImage(result => {
            dragSource.Drag.imageSource = result.url;
            dragSource.Drag.startDrag();
            DragService.end();
        });
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
            cutSet: window.cutSet
            onActivateRequested: entry => window.activate(entry)
            onItemMenuRequested: (index, pointX, pointY, modifiers) => window.showItemMenu(pointX, pointY, modifiers)
            onBackgroundMenuRequested: (pointX, pointY) => window.showBackgroundMenu(pointX, pointY)
            onDropRequested: (paths, dest, action) => window.applyDrop(paths, dest, action)
            onDragRequested: paths => window.beginDrag(paths)
            onRenameRequested: (target, name) => FileOpsService.rename(target, name, window.opScope)
            onSortRequested: key => window.setSort(key)
            onColumnsMenuRequested: (pointX, pointY) => window.showColumnsMenu(pointX, pointY)
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
                onDropRequested: (path, drop) => window.applyDrop(DragService.pathsFrom(drop), path, DragService.actionFor(drop.modifiers ?? Qt.NoModifier))
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
                onTabDropHovered: index => window.selectTab(index)
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
                viewMode: SessionData.viewMode
                onSidebarToggled: window.toggleSidebar()
                onBackRequested: window.activePane.back()
                onForwardRequested: window.activePane.forward()
                onSplitToggled: window.toggleSplit()
                onBookmarkToggled: BookmarksService.toggle(window.currentPath)
                onSortCycled: window.cycleSort()
                onSortDirectionToggled: window.sortDescending = !window.sortDescending
                onCopyPathRequested: Paths.copyPathToClipboard(window.currentPath)
                onViewModeSelected: mode => SessionData.viewMode = mode
                onReloadRequested: window.runAction("reload")
                onTerminalRequested: window.openTerminalHere()
                onPropertiesRequested: window.runAction("properties")
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
                    cutSet: window.cutSet
                    onActivateRequested: entry => window.activate(entry)
                    onItemMenuRequested: (index, pointX, pointY, modifiers) => window.showItemMenu(pointX, pointY, modifiers)
                    onBackgroundMenuRequested: (pointX, pointY) => window.showBackgroundMenu(pointX, pointY)
                    onDropRequested: (paths, dest, action) => window.applyDrop(paths, dest, action)
                    onDragRequested: paths => window.beginDrag(paths)
                    onRenameRequested: (target, name) => FileOpsService.rename(target, name, window.opScope)
                    onSortRequested: key => window.setSort(key)
                    onColumnsMenuRequested: (pointX, pointY) => window.showColumnsMenu(pointX, pointY)
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
                onDropRequested: (path, drop) => window.applyDrop(DragService.pathsFrom(drop), path, DragService.actionFor(drop.modifiers ?? Qt.NoModifier))
            }
        }
    }

    // One Shortcut per table row, so the keyboard map lives in exactly one
    // place and every binding dispatches through runAction like the menus do.
    Instantiator {
        model: Shortcuts.bound("window")

        Shortcut {
            required property var modelData

            enabled: window.shortcutsEnabled
            sequences: modelData.keys
            onActivated: window.runAction(modelData.id)
        }
    }

    Instantiator {
        model: Shortcuts.bound("selection")

        Shortcut {
            required property var modelData

            enabled: window.shortcutsEnabled && window.selectedPaths().length > 0
            sequences: modelData.keys
            onActivated: window.runAction(modelData.id)
        }
    }

    Instantiator {
        model: Shortcuts.bound("path")

        Shortcut {
            required property var modelData

            enabled: !window.dialogOpen
            sequences: modelData.keys
            onActivated: window.runAction(modelData.id)
        }
    }

    FileContextMenu {
        id: itemMenu

        anchorItem: surface
        selectionCount: window.selectedPaths().length
        hasDirectory: window.selectionIsDirectory()
        hasFile: window.selectionIsFile()
        hasClipboard: ClipboardService.hasFiles
        shiftHeld: window.shiftHeld
        onActionTriggered: id => window.runAction(id)
    }

    ContextMenu {
        id: columnsMenu

        anchorItem: surface

        Repeater {
            model: FileColumns.optional

            ContextMenuItem {
                required property var modelData

                text: FileColumns.label(modelData.id)
                iconName: SessionData.listColumns.includes(modelData.id) ? "check" : ""
                onClicked: {
                    columnsMenu.close();
                    window.toggleColumn(modelData.id);
                }
            }
        }
    }

    FileContextMenu {
        id: backgroundMenu

        anchorItem: surface
        forFolder: true
        hasClipboard: ClipboardService.hasFiles
        onActionTriggered: id => window.runAction(id)
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
                    itemMenu.close();
                    AppsService.openWith(window.selectedPaths(), modelData.id, response => window.reportError(response));
                }
            }
        }

        ContextMenuItem {
            text: I18n.tr("Other application…", "open with menu entry opening the full application picker")
            iconName: "more_horiz"
            onClicked: {
                openWithMenu.close();
                itemMenu.close();
                window.openWithCurrent();
            }
        }
    }

    // The drag payload item is never seen: the pointer carries the grabbed
    // ghost image, and the mime data is what other applications receive.
    Item {
        id: dragSource

        Drag.dragType: Drag.Automatic
        Drag.supportedActions: Qt.CopyAction | Qt.MoveAction | Qt.LinkAction
    }

    DragGhost {
        id: dragGhost

        x: -width * 2
        y: -height * 2
        action: DragService.action
    }

    PropertiesDialog {
        id: propertiesDialog

        onRenameRequested: (target, name) => FileOpsService.rename(target, name, window.opScope)
    }

    ShortcutsOverlay {
        id: shortcutsOverlay
    }

    ContextMenu {
        id: dropMenu

        property var pending: []
        property string dest: ""

        anchorItem: surface

        function present(paths, target) {
            pending = paths;
            dest = target;
            openAt(surface.width / 2 - width / 2, surface.height / 2 - height / 2);
        }

        ContextMenuItem {
            text: I18n.tr("Copy", "context menu entry copying the selection")
            iconName: "content_copy"
            onClicked: {
                dropMenu.close();
                FileOpsService.copy(dropMenu.pending, dropMenu.dest, window.opScope);
            }
        }

        ContextMenuItem {
            text: I18n.tr("Move", "drag label for moving files")
            iconName: "drive_file_move"
            onClicked: {
                dropMenu.close();
                FileOpsService.move(dropMenu.pending, dropMenu.dest, window.opScope);
            }
        }

        ContextMenuItem {
            text: I18n.tr("Link", "drag label for creating a symlink")
            iconName: "add_link"
            onClicked: {
                dropMenu.close();
                FileOpsService.link(dropMenu.pending, dropMenu.dest, window.opScope);
            }
        }

        ContextMenuItem {
            text: I18n.tr("Cancel", "dialog button to cancel")
            iconName: "close"
            onClicked: dropMenu.close()
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
