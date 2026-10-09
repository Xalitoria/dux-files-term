pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Window
import Quickshell
import Quickshell.Wayland
import qs.Common
import qs.Services
import qs.DCommon.Common as DC
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser
import qs.Modules.Settings

FloatingWindow {
    id: window

    signal closeRequested
    signal newWindowRequested(var locations)

    required property int windowId
    readonly property bool primary: WindowService.primaryId === windowId
    property var toplevel: null

    property var tabs: []
    property int currentTab: 0
    property int activePaneIndex: 0
    property bool splitView: false
    property real splitRatio: 0.5
    property bool showHidden: SettingsData.showHidden
    property string sortKey: SettingsData.defaultSort
    property bool sortDescending: SettingsData.defaultSortDescending
    property string viewMode: SettingsData.defaultView
    property bool sidebarOverlayOpen: false
    property string pendingTrust: ""
    property bool applyingTab: false
    property int propertiesRequest: 0
    property var closedTabs: []
    property string vimPrefix: ""
    property bool sidebarFocusPending: false
    property var pendingExtracts: ({})

    readonly property string opScope: "window" + windowId
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
    readonly property bool dialogOpen: conflictDialog.visible || errorDialog.visible || archivePasswordDialog.visible || confirmDialog.visible || newFolderDialog.visible || openWithDialog.visible || executableDialog.visible || trustDialog.visible || unlockDialog.visible || connectDialog.visible || credentialDialog.visible || propertiesDialog.visible || shortcutsOverlay.visible || compressDialog.visible || newDocumentDialog.visible || renameBookmarkDialog.visible || trashConfirmDialog.visible
    readonly property bool textEditing: surface.focusItem instanceof TextInput || surface.focusItem instanceof TextEdit
    readonly property bool sidebarFocused: isInside(surface.focusItem, dockedSidebar) || isInside(surface.focusItem, overlaySidebar)
    readonly property bool shortcutsEnabled: !dialogOpen && !textEditing

    title: I18n.tr("Files", "main window title")
    implicitWidth: SessionData.windowWidth
    implicitHeight: SessionData.windowHeight
    minimumSize: Qt.size(FilesMetrics.windowMinWidth, FilesMetrics.windowMinHeight)
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

    onCompactChanged: {
        if (!compact)
            sidebarOverlayOpen = false;
    }

    Component.onCompleted: {
        const initial = WindowService.takePendingLocations(windowId);
        if (primary)
            restoreSession();
        else
            startFresh();
        maximized = SessionData.maximized;
        if (primary)
            offerGtkImport();
        primaryPane.focusBody();
        if (initial.length > 0)
            openLocations(initial, true);
    }

    function claimToplevel() {
        if (!surface.windowActive)
            return;
        const active = ToplevelManager.activeToplevel;
        if (active && active.appId === "com.danklinux.dankfiles")
            toplevel = active;
    }

    function openSettings() {
        settingsLoader.active = true;
        settingsLoader.item.show();
    }

    function offerGtkImport() {
        if (!BookmarksService.importOffer)
            return;
        BookmarksService.declineImport();
        ToastService.show(I18n.tr("Import your GTK bookmarks?", "toast offering to copy the gtk file chooser bookmarks once"), {
            "actionLabel": I18n.tr("Import", "toast action importing gtk bookmarks"),
            "duration": ToastService.offerDuration,
            "action": () => BookmarksService.importGtk(added => ToastService.info(I18n.tr("%1 bookmarks imported", "toast after importing gtk bookmarks").arg(added)))
        });
    }

    function requestClose() {
        closeRequested();
    }

    function toggleSidebar() {
        if (compact) {
            sidebarOverlayOpen = !sidebarOverlayOpen;
            return;
        }
        SettingsData.sidebarCollapsed = !SettingsData.sidebarCollapsed;
    }

    function isInside(item, ancestor) {
        for (let current = item; current; current = current.parent) {
            if (current === ancestor)
                return true;
        }
        return false;
    }

    function focusSidebar() {
        if (compact)
            sidebarOverlayOpen = true;
        else
            SettingsData.sidebarCollapsed = false;
        const sidebar = compact ? overlaySidebar : dockedSidebar;
        const shown = compact ? sidebarOverlay.visible : dockedSidebar.visible;
        if (!shown) {
            sidebarFocusPending = true;
            return;
        }
        sidebar.focusCurrent();
    }

    function sidebarShown(sidebar) {
        if (!sidebarFocusPending)
            return;
        sidebarFocusPending = false;
        sidebar.focusCurrent();
    }

    function leaveSidebar() {
        sidebarOverlayOpen = false;
        activePane.focusBody();
    }

    function renameBookmark(path) {
        renameBookmarkDialog.path = path;
        renameBookmarkDialog.ask(BookmarksService.labelFor(path));
    }

    function setSplitRatio(value) {
        splitRatio = Math.max(FilesMetrics.minimumPaneFraction, Math.min(1 - FilesMetrics.minimumPaneFraction, value));
        if (primary)
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
        persistTabs();
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
        persistTabs();
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
        persistTabs();
    }

    function closeTab(index) {
        if (tabs.length <= 1)
            return;
        captureTabs();
        closedTabs = closedTabs.concat([
            {
                "index": index,
                "state": tabs[index]
            }
        ]).slice(-FilesMetrics.closedTabHistory);
        const next = tabs.slice();
        next.splice(index, 1);
        const target = index < currentTab ? currentTab - 1 : Math.min(currentTab, next.length - 1);
        tabs = next;
        currentTab = target;
        applyTab(target);
        persistTabs();
    }

    function cycleTab(step) {
        if (tabs.length < 2)
            return;
        selectTab((currentTab + step + tabs.length) % tabs.length);
    }

    function reopenTab() {
        if (closedTabs.length === 0)
            return;
        const stack = closedTabs.slice();
        const closed = stack.pop();
        closedTabs = stack;
        captureTabs();
        const next = tabs.slice();
        const index = Math.min(closed.index, next.length);
        next.splice(index, 0, closed.state);
        tabs = next;
        currentTab = index;
        applyTab(index);
        persistTabs();
    }

    function tabLabel(state) {
        const path = state?.panes?.[0]?.path ?? "";
        if (path === "" || path === Paths.strip(Paths.home))
            return I18n.tr("Home", "sidebar entry for the user's home folder");
        if (path === "/")
            return "/";
        return path.substring(path.lastIndexOf("/") + 1);
    }

    function persistTabs() {
        if (!primary)
            return;
        SessionData.tabs = tabs;
        SessionData.currentTab = currentTab;
    }

    function homeTabs() {
        return [
            {
                "panes": [
                    {
                        "path": Paths.strip(Paths.home)
                    },
                    null],
                "active": 0
            }
        ];
    }

    function restoreSession() {
        splitView = SessionData.splitView;
        splitRatio = SessionData.splitRatio;
        const stored = SessionData.tabs;
        tabs = Array.isArray(stored) && stored.length > 0 ? stored : homeTabs();
        currentTab = Math.max(0, Math.min(SessionData.currentTab, tabs.length - 1));
        applyTab(currentTab);
    }

    function startFresh() {
        splitRatio = SessionData.splitRatio;
        tabs = homeTabs();
        currentTab = 0;
        applyTab(0);
    }

    function folderLocation(path) {
        return {
            "kind": "folder",
            "path": path,
            "select": []
        };
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

    function openLocations(locations, fresh) {
        locations.forEach((location, index) => openLocation(location, index > 0 || (!fresh && SettingsData.openLocationIn === "tab")));
    }

    function openLocation(location, inNewTab) {
        switch (location.kind) {
        case "servers":
            connectDialog.ask();
            return;
        case "remote":
            connectRemote(location.uri);
            return;
        case "archive":
            openArchive(location.path, inNewTab);
            return;
        }
        if (location.select.length === 0) {
            openPath(location.path, inNewTab);
            return;
        }
        if (inNewTab)
            newTab(location.path);
        activePane.reveal(location.select);
    }

    function openArchive(path, inNewTab) {
        openPath(FilePaths.parentOf(path), inNewTab);
        if (!SettingsData.extractOnOpen) {
            activePane.reveal([path]);
            return;
        }
        extractBeside(path, "enter");
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
        if (primary)
            SessionData.splitView = splitView;
        if (!splitView) {
            activePaneIndex = 0;
            primaryPane.focusBody();
            return;
        }
        Qt.callLater(() => {
            if (!window.secondPane)
                return;
            const stored = window.tabState(window.currentTab)?.panes?.[1];
            window.secondPane.restoreState(stored ?? {
                "path": primaryPane.path
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

    function selectSortKey(key) {
        if (key !== "")
            sortKey = key;
    }

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
            newTab("");
            return;
        case "newWindow":
            newWindowRequested([folderLocation(currentPath)]);
            return;
        case "openInNewWindow":
            openSelectionInNewWindow();
            return;
        case "openInNewTab":
            openSelectionInNewTab();
            return;
        case "reopenTab":
            reopenTab();
            return;
        case "home":
            openPath(Paths.strip(Paths.home), false);
            return;
        case "splitView":
            toggleSplit();
            return;
        case "toggleSidebar":
            toggleSidebar();
            return;
        case "focusSidebar":
            focusSidebar();
            return;
        case "bookmark":
            BookmarksService.toggle(currentPath);
            return;
        case "zoomReset":
            activePane.setZoom(viewMode, FilesMetrics.defaultZoom(viewMode));
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
        case "settings":
            openSettings();
            return;
        case "compress":
            compressDialog.ask(selectedPaths());
            return;
        case "extract":
            extractTo(selectedPaths(), currentPath, "reveal");
            return;
        case "extractTo":
            askExtractTo(selectedPaths());
            return;
        case "newDocument":
            showNewDocumentMenu();
            return;
        case "swapPane":
            swapPaneFocus();
            return;
        case "listView":
            viewMode = "list";
            return;
        case "gridView":
            viewMode = "grid";
            return;
        case "columnsView":
            viewMode = "columns";
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
            if (shortcutsOverlay.visible) {
                shortcutsOverlay.close();
                return;
            }
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
            Paths.copyPathToClipboard((selectedPaths().length > 0 ? selectedPaths() : [currentPath]).join("\n"));
            return;
        case "contextMenu":
            showContextMenuAtCursor();
            return;
        }
    }

    function openSelectionInNewTab() {
        const entry = firstSelected();
        if (entry?.isDir)
            newTab(entry.path);
    }

    function openSelectionInNewWindow() {
        const entry = firstSelected();
        if (entry?.isDir)
            newWindowRequested([folderLocation(entry.path)]);
    }

    function showContextMenuAtCursor() {
        if (selectedPaths().length === 0) {
            showBackgroundMenu(activePane.width / 2, activePane.height / 2);
            return;
        }
        const rect = activePane.cursorRect();
        if (rect.width === 0) {
            showItemMenu(activePane.width / 2, activePane.height / 2, Qt.NoModifier);
            return;
        }
        showItemMenu(rect.x + Math.min(rect.width, rect.height), rect.y + rect.height, Qt.NoModifier);
    }

    function escapeBack() {
        if (sidebarOverlayOpen) {
            leaveSidebar();
            return true;
        }
        if (!activePane.resultsActive && activePane.nameFilter === "")
            return false;
        topbar.clearSearch();
        return true;
    }

    function handleVimKey(event) {
        if (!SettingsData.vimKeys || (event.modifiers & ~(Qt.ShiftModifier | Qt.KeypadModifier)) !== 0)
            return false;
        const shift = (event.modifiers & Qt.ShiftModifier) !== 0;
        const prefix = vimPrefix;
        vimPrefix = "";
        if (prefix === "g")
            return runPrefixedVimKey(event.key, shift);
        const body = activePane.body;
        const columns = body.view?.columnsPerRow ?? 1;
        switch (event.key) {
        case Qt.Key_J:
            body.moveCursor(columns, event.modifiers);
            return true;
        case Qt.Key_K:
            body.moveCursor(-columns, event.modifiers);
            return true;
        case Qt.Key_H:
            if (shift)
                activePane.back();
            else
                activePane.up();
            return true;
        case Qt.Key_L:
            if (shift)
                activePane.forward();
            else
                activePane.openSelection();
            return true;
        case Qt.Key_G:
            if (shift) {
                body.jumpTo(body.directory.count - 1, Qt.NoModifier);
                return true;
            }
            vimPrefix = "g";
            vimPrefixTimer.restart();
            return true;
        case Qt.Key_V:
            body.toggleAtCursor();
            return true;
        case Qt.Key_Y:
            return runOnSelection("copy");
        case Qt.Key_X:
            return runOnSelection("cut");
        case Qt.Key_P:
            runAction("paste");
            return true;
        case Qt.Key_D:
            return runOnSelection(shift ? "deletePermanently" : "trash");
        case Qt.Key_R:
            return runOnSelection("rename");
        case Qt.Key_U:
            runAction("undo");
            return true;
        case Qt.Key_Period:
            runAction("showHidden");
            return true;
        case Qt.Key_T:
            runAction("newTab");
            return true;
        case Qt.Key_S:
            runAction("focusSidebar");
            return true;
        }
        return false;
    }

    function runPrefixedVimKey(key, shift) {
        switch (key) {
        case Qt.Key_G:
            if (shift)
                return false;
            activePane.body.jumpTo(0, Qt.NoModifier);
            return true;
        case Qt.Key_T:
            cycleTab(shift ? -1 : 1);
            return true;
        case Qt.Key_H:
            if (shift)
                return false;
            runAction("home");
            return true;
        }
        return false;
    }

    function runOnSelection(id) {
        if (selectedPaths().length > 0)
            runAction(id);
        return true;
    }

    function openInOtherPane(path) {
        if (path === "")
            return;
        if (!splitView)
            toggleSplit();
        Qt.callLater(() => {
            const target = window.activePaneIndex === 0 ? window.secondPane : primaryPane;
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
        if (canRun && FilesFormat.isRunnable(entry.mime, entry.isExecutable)) {
            executableDialog.ask(entry.path, entry.name);
            return;
        }
        if (SettingsData.extractOnOpen && FilesFormat.extractable(entry.mime)) {
            extractBeside(entry.path, "enter");
            return;
        }
        AppsService.open([entry.path], response => window.reportError(response));
    }

    function extractBeside(path, follow) {
        extractTo([path], FilePaths.parentOf(path), follow);
    }

    function extractTo(paths, dest, follow) {
        const pending = pendingExtracts;
        for (const path of paths)
            pending[path] = follow;
        pendingExtracts = pending;
        FileOpsService.extract(paths, dest, opScope);
    }

    function askExtractTo(paths) {
        if (paths.length === 0)
            return;
        extractPickerLoader.active = true;
        extractPickerLoader.item.archives = paths;
        extractPickerLoader.item.open();
    }

    function takeExtractFollow(sources) {
        const pending = pendingExtracts;
        let follow = "reveal";
        for (const source of sources) {
            if (pending[source])
                follow = pending[source];
            delete pending[source];
        }
        pendingExtracts = pending;
        return follow;
    }

    function handleExtracted(job) {
        const follow = takeExtractFollow(job.sources);
        const created = job.created ?? [];
        if (job.cancelled || created.length === 0)
            return;
        const paths = created.map(entry => entry.path);
        const dest = FilePaths.parentOf(paths[0]);
        const stillHere = dest === activePane.path && !activePane.resultsActive;
        if (stillHere && follow === "enter" && created.length === 1 && created[0].dir) {
            activePane.navigate(paths[0]);
            undoToast(I18n.tr("Extracted", "toast after unpacking an archive"));
            return;
        }
        if (stillHere) {
            activePane.reveal(paths);
            undoToast(I18n.tr("Extracted", "toast after unpacking an archive"));
            return;
        }
        ToastService.show(I18n.tr("Extracted to %1", "toast after unpacking into another folder, %1 is the folder name").arg(FilePaths.baseName(dest)), {
            "actionLabel": I18n.tr("Show", "toast action revealing the extracted files"),
            "action": () => activePane.reveal(paths)
        });
    }

    function undoToast(message) {
        ToastService.show(message, {
            "actionLabel": I18n.tr("Undo", "toast action reversing the last operation"),
            "action": () => FileOpsService.undo(opScope)
        });
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
        activePane.reveal(entry.path);
    }

    function selectionIsExtractable() {
        const entries = selectedEntries();
        return entries.length > 0 && entries.every(entry => !entry.isDir && FilesFormat.extractable(entry.mime));
    }

    function showNewDocumentMenu() {
        newDocumentMenu.templates = [];
        const dir = DirectoryService.templatesDir;
        if (dir !== "") {
            DirectoryService.list(dir, {
                "limit": 100
            }, result => {
                if (result.error)
                    return;
                newDocumentMenu.templates = (result.entries || []).filter(entry => !entry.isDir);
            });
        }
        openWithMenu.close();
        itemMenu.close();
        newDocumentMenu.openAt(backgroundMenu.x, backgroundMenu.y);
    }

    function createDocument(template) {
        newDocumentDialog.template = template?.path ?? "";
        newDocumentDialog.ask(template ? template.name : I18n.tr("Untitled document", "default name for a new empty file"));
    }

    function showBookmarkMenu(path, source, pointX, pointY) {
        bookmarkMenu.path = path;
        const point = source.mapToItem(surface, pointX, pointY);
        bookmarkMenu.openAt(point.x, point.y);
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
        if (!SettingsData.confirmTrash) {
            FileOpsService.trash(paths, opScope);
            return;
        }
        trashConfirmDialog.pending = paths;
        trashConfirmDialog.subject = paths.length === 1 ? firstSelected().name : FileFormat.count(paths.length);
        trashConfirmDialog.open();
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
        showPropertiesFor([currentPath]);
    }

    function connectRemote(uri) {
        NetworkService.connectTo(uri, result => {
            const path = connectDialog.localPath(result);
            if (path === "") {
                connectDialog.showFailure(uri, result);
                return;
            }
            window.openPath(path, true);
        });
    }

    function showPropertiesFor(paths) {
        const request = ++propertiesRequest;
        const entries = [];
        let pending = paths.length;
        paths.forEach((path, index) => DankFilesService.sendRequest("files.stat", {
                "path": path
            }, response => {
                if (request !== window.propertiesRequest)
                    return;
                pending--;
                if (response.error)
                    ToastService.info(FileFormat.operationError(response.code, response.error));
                entries[index] = (response.result || {}).entry ?? null;
                if (pending > 0)
                    return;
                const found = entries.filter(entry => entry);
                if (found.length > 0)
                    propertiesDialog.show(found);
            }));
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
        if (SettingsData.dragAction !== "auto") {
            applyDrop(paths, dest, SettingsData.dragAction);
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
        case "compress":
            return I18n.tr("Archive created", "toast after compressing files");
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
        target: BookmarksService

        function onRequestFailed(message) {
            ToastService.info(message);
        }

        function onImportOfferChanged() {
            window.offerGtkImport();
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
                if (job.kind === "extract")
                    window.takeExtractFollow(job.sources);
                ToastService.info(FileFormat.operationError(job.code, job.error));
                return;
            }
            DevicesService.refreshUsage(true);
            if (job.scope !== window.opScope)
                return;
            if (job.kind === "extract") {
                window.handleExtracted(job);
                return;
            }
            if (job.cancelled || !job.undoable)
                return;
            const message = window.toastFor(job);
            if (message === "")
                return;
            window.undoToast(message);
        }
    }

    Timer {
        id: vimPrefixTimer

        interval: FileBrowserMetrics.typeAheadInterval
        onTriggered: window.vimPrefix = ""
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
            viewMode: window.viewMode
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
            onPathChanged: {
                window.captureTabs();
                SessionData.recordRecent(path);
            }
        }
    }

    Item {
        id: surface

        anchors.fill: parent
        readonly property bool isFloatingWindowSurface: true
        readonly property bool windowActive: Window.active
        readonly property Item focusItem: Window.activeFocusItem

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        WheelHandler {
            property var forwardedScroller: null

            acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
            acceptedModifiers: Qt.NoModifier
            enabled: !window.dialogOpen
            onWheel: event => {
                const point = sidebarSlot.mapFromItem(surface, event.x, event.y);
                const overSidebar = window.sidebarDocked && point.x >= 0 && point.x <= sidebarSlot.width && point.y >= 0;
                forwardedScroller = overSidebar ? dockedSidebar : window.activePane.scroller;
                if (!forwardedScroller?.forwardWheel) {
                    event.accepted = false;
                    return;
                }
                forwardedScroller.forwardWheel(event);
            }
            onActiveChanged: {
                if (active)
                    return;
                forwardedScroller?.forwardWheelEnd();
                forwardedScroller = null;
            }
        }

        onWindowActiveChanged: {
            if (!windowActive)
                return;
            WindowService.activate(window.windowId);
            window.claimToplevel();
            primaryPane.refresh();
            window.secondPane?.refresh();
        }

        Keys.onPressed: event => {
            if (window.textEditing)
                return;
            if (!window.sidebarFocused && !window.isInside(surface.focusItem, window.activePane)) {
                window.activePane.handleStrayKey(event);
                if (event.accepted)
                    return;
            }
            switch (event.key) {
            case Qt.Key_Question:
                window.runAction("shortcuts");
                break;
            case Qt.Key_Slash:
                window.runAction("search");
                break;
            case Qt.Key_Escape:
                if (!window.escapeBack())
                    return;
                break;
            default:
                if (!window.handleVimKey(event))
                    return;
            }
            event.accepted = true;
        }

        DWindowHeader {
            id: header

            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            controls: windowControls
            title: window.title
            onCloseRequested: window.requestClose()

            DActionButton {
                id: jobsButton

                anchors.verticalCenter: parent.verticalCenter
                visible: FileOpsService.jobs.length > 0
                iconName: FileOpsService.activeCount > 0 ? "sync" : "check_circle"
                buttonSize: Theme.buttonHeightXXS
                iconSize: Theme.iconSizeSmall
                iconColor: Theme.surfaceText
                tooltipText: I18n.tr("File operations", "jobs popover header")
                onClicked: jobsPopover.visible ? jobsPopover.close() : jobsPopover.open()

                JobsPopover {
                    id: jobsPopover

                    y: jobsButton.height + Theme.spacingXS
                    x: jobsButton.width - width
                }
            }
        }

        Item {
            id: sidebarSlot

            readonly property real maximumWidth: Math.max(FilesMetrics.sidebarMinWidth, surface.width - FilesMetrics.windowMinWidth)
            property real preferredWidth: CacheData.sidebarWidth

            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            width: window.sidebarDocked ? Math.max(FilesMetrics.sidebarMinWidth, Math.min(maximumWidth, preferredWidth)) : 0
            clip: true

            Connections {
                target: CacheData

                function onSidebarWidthChanged() {
                    if (!sidebarResizeArea.pressed)
                        sidebarSlot.preferredWidth = CacheData.sidebarWidth;
                }
            }

            Behavior on width {
                enabled: Theme.animationsEnabled && !sidebarResizeArea.pressed
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
                onVisibleChanged: {
                    if (visible)
                        window.sidebarShown(dockedSidebar);
                }
                onNavigateRequested: path => window.openPath(path, false)
                onUnlockRequested: device => unlockDialog.ask(device)
                onConnectRequested: connectDialog.ask()
                onBookmarkMenuRequested: (path, pointX, pointY) => window.showBookmarkMenu(path, dockedSidebar, pointX, pointY)
                onBookmarkRenameRequested: path => window.renameBookmark(path)
                onLeaveRequested: window.leaveSidebar()
                onDropRequested: (path, drop) => window.applyDrop(DragService.pathsFrom(drop), path, DragService.actionFor(drop.modifiers ?? Qt.NoModifier))
            }
        }

        Item {
            id: sidebarResizer

            anchors.left: sidebarSlot.right
            anchors.leftMargin: -width / 2
            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            width: FilesMetrics.splitterWidth
            visible: window.sidebarDocked
            z: 1

            Rectangle {
                anchors.centerIn: parent
                width: Theme.outlineWidthFocused
                height: FilesMetrics.splitterHandleHeight
                radius: Theme.cornerRadiusFull
                color: Theme.primary
                visible: sidebarResizeArea.containsMouse || sidebarResizeArea.pressed
            }

            MouseArea {
                id: sidebarResizeArea

                property real pressX: 0
                property real startWidth: 0

                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.SplitHCursor
                acceptedButtons: Qt.LeftButton
                onPressed: mouse => {
                    pressX = mapToItem(surface, mouse.x, 0).x;
                    startWidth = sidebarSlot.width;
                }
                onPositionChanged: mouse => {
                    if (!pressed)
                        return;
                    const delta = mapToItem(surface, mouse.x, 0).x - pressX;
                    const width = startWidth + (I18n.isRtl ? -delta : delta);
                    sidebarSlot.preferredWidth = Math.max(FilesMetrics.sidebarMinWidth, Math.min(sidebarSlot.maximumWidth, width));
                }
                onReleased: CacheData.sidebarWidth = Math.round(sidebarSlot.preferredWidth)
                onCanceled: sidebarSlot.preferredWidth = CacheData.sidebarWidth
                onDoubleClicked: {
                    sidebarSlot.preferredWidth = FilesMetrics.sidebarWidth;
                    CacheData.sidebarWidth = Math.round(FilesMetrics.sidebarWidth);
                }
            }
        }

        Item {
            id: content

            anchors.top: header.bottom
            anchors.bottom: parent.bottom
            anchors.left: sidebarSlot.right
            anchors.right: parent.right
            anchors.topMargin: FilesMetrics.contentMarginV
            anchors.bottomMargin: FilesMetrics.contentMarginV
            anchors.leftMargin: FilesMetrics.contentMarginH
            anchors.rightMargin: FilesMetrics.contentMarginH

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
                compact: window.compact
                viewMode: window.viewMode
                filterText: window.activePane.nameFilter
                currentPath: window.currentPath
                recentDirs: SessionData.recentDirs
                onFilterChanged: text => window.activePane.nameFilter = text
                onSearchRequested: query => window.activePane.startSearch(query, null)
                onSearchCleared: {
                    window.activePane.closeSearch();
                    window.activePane.nameFilter = "";
                    window.activePane.focusBody();
                }
                onNavigateRequested: path => {
                    window.openPath(path, false);
                    window.activePane.focusBody();
                }
                onSettingsRequested: window.openSettings()
                onShortcutsRequested: window.runAction("shortcuts")
                onSidebarToggled: window.toggleSidebar()
                onBackRequested: window.activePane.back()
                onForwardRequested: window.activePane.forward()
                onSplitToggled: window.toggleSplit()
                onBookmarkToggled: BookmarksService.toggle(window.currentPath)
                onSortKeySelected: key => window.selectSortKey(key)
                onSortDirectionToggled: window.sortDescending = !window.sortDescending
                onCopyPathRequested: Paths.copyPathToClipboard(window.currentPath)
                onViewModeSelected: mode => window.viewMode = mode
                onReloadRequested: window.runAction("reload")
                onTerminalRequested: window.openTerminalHere()
                onPropertiesRequested: window.runAction("properties")
                onNewWindowRequested: window.runAction("newWindow")
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
                    viewMode: window.viewMode
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
                    onPathChanged: {
                        window.captureTabs();
                        SessionData.recordRecent(path);
                    }
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
            width: Math.max(FilesMetrics.sidebarMinWidth, Math.min(window.width - FilesMetrics.contentMarginH * 2, CacheData.sidebarWidth))
            visible: x > -width
            x: window.sidebarOverlayOpen ? 0 : -width
            color: Theme.surfaceContainerLow
            onVisibleChanged: {
                if (visible)
                    window.sidebarShown(overlaySidebar);
            }
            topRightRadius: Theme.windowRadius
            bottomRightRadius: Theme.windowRadius

            DC.ElevationShadow {
                anchors.fill: parent
                z: -1
                level: Theme.elevationLevel1
                targetRadius: Theme.windowRadius
                targetColor: sidebarOverlay.color
                shadowEnabled: Theme.elevationEnabled
            }

            Behavior on x {
                enabled: Theme.animationsEnabled
                NumberAnimation {
                    duration: Theme.expressiveDurations.expressiveDefaultSpatial
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Theme.expressiveCurves.expressiveDefaultSpatial
                }
            }

            Sidebar {
                id: overlaySidebar

                anchors.fill: parent
                anchors.margins: FilesMetrics.sidebarPadding
                currentPath: window.currentPath
                onNavigateRequested: path => {
                    window.sidebarOverlayOpen = false;
                    window.openPath(path, false);
                }
                onUnlockRequested: device => unlockDialog.ask(device)
                onConnectRequested: connectDialog.ask()
                onBookmarkMenuRequested: (path, pointX, pointY) => window.showBookmarkMenu(path, overlaySidebar, pointX, pointY)
                onBookmarkRenameRequested: path => window.renameBookmark(path)
                onLeaveRequested: window.leaveSidebar()
                onDropRequested: (path, drop) => window.applyDrop(DragService.pathsFrom(drop), path, DragService.actionFor(drop.modifiers ?? Qt.NoModifier))
            }
        }
    }

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

            enabled: window.shortcutsEnabled && !window.sidebarFocused && window.selectedPaths().length > 0
            sequences: modelData.keys
            onActivated: window.runAction(modelData.id)
        }
    }

    Instantiator {
        model: FilesMetrics.tabShortcutCount

        Shortcut {
            required property int index

            enabled: window.shortcutsEnabled
            sequence: "Alt+" + (index + 1)
            onActivated: window.selectTab(index)
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
        extractable: window.selectionIsExtractable()
        inResults: window.activePane.resultsActive
        onActionTriggered: id => window.runAction(id)
    }

    FileMenu {
        id: newDocumentMenu

        property var templates: []

        anchorItem: surface

        FileMenuItem {
            text: I18n.tr("Empty file", "new document menu entry creating a blank file")
            iconName: "draft"
            onClicked: {
                newDocumentMenu.close();
                window.createDocument(null);
            }
        }

        Repeater {
            model: newDocumentMenu.templates

            FileMenuItem {
                required property var modelData

                text: modelData.displayName !== "" ? modelData.displayName : modelData.name
                iconName: "description"
                onClicked: {
                    newDocumentMenu.close();
                    window.createDocument(modelData);
                }
            }
        }
    }

    FileMenu {
        id: bookmarkMenu

        property string path: ""

        anchorItem: surface

        FileMenuItem {
            text: I18n.tr("Open in new tab", "context menu entry opening a folder in a new tab")
            iconName: "tab"
            onClicked: {
                bookmarkMenu.close();
                window.newTab(bookmarkMenu.path);
            }
        }

        FileMenuItem {
            text: I18n.tr("Open in new window", "context menu entry opening a folder in another window")
            iconName: "open_in_new"
            onClicked: {
                bookmarkMenu.close();
                window.newWindowRequested([window.folderLocation(bookmarkMenu.path)]);
            }
        }

        FileMenuItem {
            text: I18n.tr("Rename", "rename dialog header")
            iconName: "edit"
            onClicked: {
                bookmarkMenu.close();
                window.renameBookmark(bookmarkMenu.path);
            }
        }

        FileMenuItem {
            text: I18n.tr("Remove bookmark", "tooltip on the bookmark button")
            iconName: "bookmark_remove"
            onClicked: {
                bookmarkMenu.close();
                BookmarksService.remove(bookmarkMenu.path);
            }
        }
    }

    FileMenu {
        id: columnsMenu

        anchorItem: surface

        Repeater {
            model: FileColumns.optional

            FileMenuItem {
                required property var modelData

                text: FileColumns.label(modelData.id)
                checked: SessionData.listColumns.includes(modelData.id)
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

    FileMenu {
        id: openWithMenu

        property var apps: []

        anchorItem: surface

        Repeater {
            model: openWithMenu.apps

            FileMenuItem {
                required property var modelData

                text: modelData.name
                checked: modelData.isDefault === true
                onClicked: {
                    openWithMenu.close();
                    itemMenu.close();
                    AppsService.openWith(window.selectedPaths(), modelData.id, response => window.reportError(response));
                }
            }
        }

        FileMenuItem {
            text: I18n.tr("Other application…", "open with menu entry opening the full application picker")
            iconName: "more_horiz"
            onClicked: {
                openWithMenu.close();
                itemMenu.close();
                window.openWithCurrent();
            }
        }
    }

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

    FileMenu {
        id: dropMenu

        property var pending: []
        property string dest: ""

        anchorItem: surface

        function present(paths, target) {
            pending = paths;
            dest = target;
            openAt(surface.width / 2 - width / 2, surface.height / 2 - height / 2);
        }

        FileMenuItem {
            text: I18n.tr("Copy", "context menu entry copying the selection")
            iconName: "content_copy"
            onClicked: {
                dropMenu.close();
                FileOpsService.copy(dropMenu.pending, dropMenu.dest, window.opScope);
            }
        }

        FileMenuItem {
            text: I18n.tr("Move", "drag label for moving files")
            iconName: "drive_file_move"
            onClicked: {
                dropMenu.close();
                FileOpsService.move(dropMenu.pending, dropMenu.dest, window.opScope);
            }
        }

        FileMenuItem {
            text: I18n.tr("Link", "drag label for creating a symlink")
            iconName: "add_link"
            onClicked: {
                dropMenu.close();
                FileOpsService.link(dropMenu.pending, dropMenu.dest, window.opScope);
            }
        }

        FileMenuItem {
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

    ArchivePasswordDialog {
        id: archivePasswordDialog
    }

    NameDialog {
        id: newFolderDialog

        title: I18n.tr("New folder", "new folder dialog header")
        acceptText: I18n.tr("Create", "dialog button creating an item")
        onSubmitted: name => FileOpsService.newFolder(window.currentPath, name, window.opScope)
    }

    NameDialog {
        id: newDocumentDialog

        property string template: ""

        title: I18n.tr("New document", "context menu entry creating a file from a template")
        acceptText: I18n.tr("Create", "dialog button creating an item")
        onSubmitted: name => FileOpsService.newFile(window.currentPath, name, template, window.opScope)
    }

    NameDialog {
        id: renameBookmarkDialog

        property string path: ""

        title: I18n.tr("Rename", "rename dialog header")
        acceptText: I18n.tr("Rename", "rename dialog header")
        onSubmitted: name => BookmarksService.rename(path, name)
    }

    CompressDialog {
        id: compressDialog

        onSubmitted: (paths, name, format, password) => FileOpsService.compress(paths, window.currentPath, name, format, password, window.opScope)
    }

    ConfirmDialog {
        id: trashConfirmDialog

        property var pending: []

        title: I18n.tr("Move to Trash", "context menu entry trashing the selection")
        message: I18n.tr("%1 will be moved to the trash.", "trash confirmation explanation").arg(subject)
        confirmText: I18n.tr("Move to Trash", "context menu entry trashing the selection")
        danger: false
        onConfirmed: {
            FileOpsService.trash(pending, window.opScope);
            pending = [];
        }
    }

    LazyLoader {
        id: extractPickerLoader

        active: false

        FileBrowserModal {
            property var archives: []

            parentModal: window
            browserTitle: I18n.tr("Extract to", "folder picker title for unpacking archives")
            bucket: "extract"
            mode: "openFolder"
            revealPath: window.currentPath
            onAccepted: paths => window.extractTo(archives, paths[0], "reveal")
        }
    }

    Loader {
        id: settingsLoader

        active: false
        sourceComponent: SettingsModal {
            parentWindow: window
        }
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

        onConnected: path => window.openPath(path, false)
    }

    CredentialDialog {
        id: credentialDialog
    }

    Connections {
        target: ToplevelManager
        function onActiveToplevelChanged() {
            window.claimToplevel();
        }
    }

    Toast {
        anchors.fill: surface
        shown: WindowService.lastActiveId === window.windowId
    }
}
