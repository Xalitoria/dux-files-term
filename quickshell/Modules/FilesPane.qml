pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

FocusScope {
    id: pane

    property bool active: true
    property bool showActiveOutline: false
    property bool showHidden: false
    property string sortKey: "name"
    property bool sortDescending: false
    property string viewMode: "list"
    property bool previewOpen: true
    property var cutSet: ({})
    property string nameFilter: ""
    property var filters: []
    property bool multiSelect: true
    property bool escapeClearsSelection: true
    property bool typeAhead: !SettingsData.vimKeys
    property int connectRequest: 0

    readonly property alias path: browser.path
    readonly property alias canGoBack: browser.canGoBack
    readonly property alias canGoForward: browser.canGoForward
    readonly property bool canGoUp: browser.canGoUp && !trashActive
    readonly property alias editingPath: browser.editingPath
    readonly property alias renaming: browser.renaming
    readonly property alias searchModel: searchModel
    readonly property bool resultsActive: searchModel.active
    readonly property bool trashActive: !resultsActive && TrashService.filesPath !== "" && browser.path === TrashService.filesPath
    readonly property var trashView: trashActive ? trashLoader.item : null
    readonly property var scroller: body.view?.activeList ?? body.view
    readonly property var selection: resultsActive ? results.selection : trashView ? trashView.selection : browser.selection
    readonly property var body: resultsActive ? results.body : trashView ? trashView.body : browser.body
    readonly property var currentEntry: resultsActive ? results.body.cursorEntry() : trashView ? trashView.body.cursorEntry() : browser.currentEntry
    readonly property var selectedEntries: resultsActive ? results.body.selectedEntries() : trashView ? trashView.body.selectedEntries() : browser.selectedEntries
    readonly property bool filtering: nameFilter !== "" && !resultsActive

    signal activateRequested(var entry)
    signal openRequested(var entries)
    signal tabRequested(string path)
    signal pathMenuRequested(string path, real pointX, real pointY)
    signal itemMenuRequested(int index, real pointX, real pointY, int modifiers)
    signal backgroundMenuRequested(real pointX, real pointY)
    signal dropRequested(var paths, string dest, string action)
    signal dragRequested(var paths)
    signal renameRequested(string target, string name)
    signal sortRequested(string key)
    signal columnsMenuRequested(real pointX, real pointY)
    signal previewToggleRequested
    signal focusRequested
    signal locationRequested(var location)
    signal restoreRequested
    signal emptyTrashRequested
    signal filterRequested(string text)
    signal filterFocusRequested

    function navigate(target) {
        closeSearch();
        browser.navigate(target);
    }

    function reveal(targets) {
        closeSearch();
        browser.reveal(targets);
    }

    function revealAndRename(target) {
        closeSearch();
        browser.revealAndRename(target);
    }

    function back() {
        closeSearch();
        browser.back();
    }

    function forward() {
        closeSearch();
        browser.forward();
    }

    function up() {
        if (trashActive)
            return;
        closeSearch();
        browser.up();
    }

    function refresh() {
        if (resultsActive) {
            searchModel.refresh();
            return;
        }
        if (trashView) {
            trashView.refresh();
            return;
        }
        browser.refresh();
    }

    function editPath() {
        closeSearch();
        browser.editPath();
    }

    function startJump() {
        closeSearch();
        browser.startJump();
    }

    function settleLocation(target) {
        browser.settleLocation(target);
    }

    function dismissPathEditor(x, y) {
        browser.dismissPathEditor(x, y);
    }

    function focusBody() {
        if (resultsActive) {
            results.focusBody();
            return;
        }
        if (trashView && !editingPath) {
            trashView.focusBody();
            return;
        }
        browser.focusBody();
    }

    function beginRename(target) {
        if (trashActive)
            return;
        if (resultsActive) {
            results.body.beginRename(target);
            return;
        }
        browser.beginRename(target);
    }

    function saveState() {
        return browser.saveState();
    }

    function restoreState(state) {
        closeSearch();
        const folder = path => path === "computer:" ? "/" : path;
        browser.restoreState(Object.assign({}, state, {
            "path": folder(state?.path),
            "back": state?.back?.map(folder),
            "forward": state?.forward?.map(folder)
        }));
    }

    function marker() {
        return browser.marker();
    }

    function openSelection() {
        const entries = selectedEntries.length > 0 ? selectedEntries : [currentEntry].filter(entry => entry);
        if (entries.length === 0)
            return false;
        openRequested(entries);
        return true;
    }

    function cursorRect() {
        const rect = body.cursorRect();
        const point = body.mapToItem(pane, rect.x, rect.y);
        return Qt.rect(point.x, point.y, rect.width, rect.height);
    }

    function startSearch(query, filterState) {
        if (path === "" || trashActive)
            return;
        const state = filterState ?? results.chips.filterState;
        searchModel.rootPath = state.scope === "folder" ? path : Paths.strip(Paths.home);
        searchModel.query = query;
        searchModel.options = SearchFilters.options(state);
        searchModel.start();
        results.focusBody();
    }

    function setSearchScope(scope) {
        results.chips.scope = scope;
    }

    function focusSearchChips() {
        if (resultsActive)
            results.chips.focusFirst();
    }

    function closeSearch() {
        if (resultsActive)
            searchModel.clear();
        results.chips.resetScope();
        nameFilter = "";
    }

    function dropOn(dest, drop) {
        const paths = DragService.pathsFrom(drop);
        if (paths.length === 0 || paths.includes(dest))
            return;
        dropRequested(paths, dest, DragService.overrideFor(drop.proposedAction));
    }

    function setZoom(mode, level) {
        if (mode === "grid") {
            SessionData.gridZoom = level;
            return;
        }
        SessionData.listZoom = level;
    }

    function setPreviewWidth(width) {
        CacheData.previewWidth = Math.round(width);
    }

    function setColumnWidth(id, width) {
        const widths = Object.assign({}, SessionData.columnWidths);
        widths[id] = width;
        SessionData.columnWidths = widths;
    }

    function forwardWheel(event) {
        if (!scroller?.forwardWheel) {
            event.accepted = false;
            return;
        }
        scroller.forwardWheel(event);
    }

    function forwardWheelEnd() {
        scroller?.forwardWheelEnd();
    }

    WheelHandler {
        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        acceptedModifiers: Qt.NoModifier
        onWheel: event => pane.forwardWheel(event)
        onActiveChanged: {
            if (!active)
                pane.forwardWheelEnd();
        }
    }

    Keys.onPressed: event => handleKey(event)

    onTrashActiveChanged: refocus.restart()
    onEditingPathChanged: {
        if (!editingPath && trashActive)
            refocus.restart();
    }

    function refocusIfOwned() {
        if (pane.active && (pane.activeFocus || !pane.Window.activeFocusItem))
            focusBody();
    }

    Timer {
        id: refocus

        interval: 0
        onTriggered: pane.refocusIfOwned()
    }

    function handleKey(event) {
        switch (event.key) {
        case Qt.Key_Return:
        case Qt.Key_Enter:
            if (event.modifiers !== Qt.NoModifier && event.modifiers !== Qt.KeypadModifier)
                return;
            if (!openSelection())
                return;
            break;
        case Qt.Key_Up:
            if ((event.modifiers & Qt.AltModifier) === 0)
                return;
            up();
            break;
        default:
            return;
        }
        event.accepted = true;
    }

    function handleStrayKey(event) {
        handleKey(event);
        if (!event.accepted)
            body.handleKey(event);
        if (event.accepted)
            focusBody();
    }

    Connections {
        target: FileOpsService

        function onJobFinished(job) {
            if (pane.resultsActive && job.state === "done")
                searchModel.refresh();
        }
    }

    SearchModel {
        id: searchModel

        backend: DirectoryService
        thumbnailsEnabled: SettingsData.thumbnails
        thumbnailSizeLimit: SettingsData.thumbnailSizeLimitMb > 0 ? SettingsData.thumbnailSizeLimitMb * 1000000 : -1
    }

    FileBrowserView {
        id: browser

        anchors.fill: parent
        visible: !pane.resultsActive && (!pane.trashActive || pane.editingPath)
        focus: !pane.resultsActive && (!pane.trashActive || pane.editingPath)
        showSidebar: false
        showNavigation: false
        showHidden: pane.showHidden
        sortKey: pane.sortKey
        sortDescending: pane.sortDescending
        viewMode: pane.viewMode
        gridZoom: SessionData.gridZoom
        listZoom: SessionData.listZoom
        listColumns: SessionData.listColumns
        columnWidths: SessionData.columnWidths
        dragEnabled: true
        dropEnabled: true
        showSelectionFooter: true
        previewDrawer: true
        previewOpen: pane.previewOpen
        previewWidth: CacheData.previewWidth
        previewKeys: true
        cutSet: pane.cutSet
        nameFilter: pane.nameFilter
        filters: pane.filters
        multiSelect: pane.multiSelect
        escapeClearsSelection: pane.escapeClearsSelection
        singleClickActivates: SettingsData.clickToOpen === "single"
        typeAheadEnabled: pane.typeAhead
        typeAheadMode: SettingsData.listTyping
        countFolders: SettingsData.folderCounts
        dirsFirst: SettingsData.foldersFirst
        thumbnailsEnabled: SettingsData.thumbnails
        thumbnailSizeLimit: SettingsData.thumbnailSizeLimitMb > 0 ? SettingsData.thumbnailSizeLimitMb * 1000000 : -1
        networkThumbnails: SettingsData.networkThumbnails
        folderJump: FolderHistory
        unlistedPath: TrashService.filesPath
        header: pane.filtering ? filterHeader : null
        pathBarTrailing: paneStatus
        emptyText: {
            if (pane.trashActive)
                return "";
            if (pane.filtering)
                return I18n.tr("No results", "search results header when nothing matched");
            return I18n.tr("This folder is empty", "empty directory placeholder");
        }
        paneColor: Theme.foregroundColor(Theme.cardSurface, true)
        paneRadius: FilesMetrics.paneRadius
        paneMargin: FilesMetrics.paneBodyMargin
        paneBorderWidth: pane.showActiveOutline && pane.active ? Theme.outlineWidthFocused : 0
        paneBorderColor: Theme.primary
        onActivateRequested: entry => pane.activateRequested(entry)
        onItemMenuRequested: (index, pointX, pointY, modifiers) => pane.itemMenuRequested(index, pointX, pointY, modifiers)
        onBackgroundMenuRequested: (pointX, pointY) => pane.backgroundMenuRequested(pointX, pointY)
        onColumnsMenuRequested: (pointX, pointY) => pane.columnsMenuRequested(pointX, pointY)
        onItemsDropped: (dest, drop) => pane.dropOn(dest, drop)
        onDragRequested: paths => pane.dragRequested(paths)
        onRenameRequested: (target, name) => pane.renameRequested(target, name)
        onSortRequested: key => pane.sortRequested(key)
        onZoomRequested: (mode, level) => pane.setZoom(mode, level)
        onColumnResizeRequested: (id, width) => pane.setColumnWidth(id, width)
        onPreviewToggleRequested: pane.previewToggleRequested()
        onPreviewResized: width => pane.setPreviewWidth(width)
        onFocusRequested: pane.focusRequested()
        onTabRequested: target => pane.tabRequested(target)
        onPathMenuRequested: (target, pointX, pointY) => pane.pathMenuRequested(target, pointX, pointY)
        onLocationRequested: location => pane.locationRequested(location)
        onLocationCancelled: pane.connectRequest++
        onFilterRequested: text => pane.filterRequested(text)
        onFilterFocusRequested: pane.filterFocusRequested()
    }

    SearchResults {
        id: results

        anchors.fill: parent
        visible: pane.resultsActive
        focus: pane.resultsActive
        model: searchModel
        viewMode: pane.viewMode
        gridZoom: SessionData.gridZoom
        listZoom: SessionData.listZoom
        columnWidths: SessionData.columnWidths
        previewOpen: pane.previewOpen
        previewWidth: CacheData.previewWidth
        cutSet: pane.cutSet
        singleClickActivates: SettingsData.clickToOpen === "single"
        typeAheadEnabled: pane.typeAhead
        countFolders: SettingsData.folderCounts
        border.width: pane.showActiveOutline && pane.active ? Theme.outlineWidthFocused : 0
        border.color: Theme.primary
        onActivateRequested: entry => pane.activateRequested(entry)
        onItemMenuRequested: (index, pointX, pointY, modifiers) => pane.itemMenuRequested(index, pointX, pointY, modifiers)
        onDragRequested: paths => pane.dragRequested(paths)
        onRenameRequested: (target, name) => pane.renameRequested(target, name)
        onZoomRequested: (mode, level) => pane.setZoom(mode, level)
        onColumnResizeRequested: (id, width) => pane.setColumnWidth(id, width)
        onPreviewToggleRequested: pane.previewToggleRequested()
        onPreviewResized: width => pane.setPreviewWidth(width)
        onFocusRequested: pane.focusRequested()
        onTabRequested: target => pane.tabRequested(target)
        onCloseRequested: {
            pane.closeSearch();
            pane.focusBody();
        }
        onFiltersChanged: pane.startSearch(searchModel.query, results.chips.filterState)
    }

    Loader {
        id: trashLoader

        anchors.fill: parent
        active: pane.trashActive
        visible: !pane.editingPath
        sourceComponent: TrashView {
            viewMode: pane.viewMode
            gridZoom: SessionData.gridZoom
            listZoom: SessionData.listZoom
            columnWidths: SessionData.columnWidths
            previewOpen: pane.previewOpen
            previewWidth: CacheData.previewWidth
            nameFilter: pane.nameFilter
            singleClickActivates: SettingsData.clickToOpen === "single"
            typeAheadEnabled: !SettingsData.vimKeys
            typeAheadMode: SettingsData.listTyping
            countFolders: SettingsData.folderCounts
            thumbnailsEnabled: SettingsData.thumbnails
            thumbnailSizeLimit: SettingsData.thumbnailSizeLimitMb > 0 ? SettingsData.thumbnailSizeLimitMb * 1000000 : -1
            outlined: pane.showActiveOutline && pane.active
            onFilterRequested: text => pane.filterRequested(text)
            onFilterFocusRequested: pane.filterFocusRequested()
            onActivateRequested: entry => pane.activateRequested(entry)
            onItemMenuRequested: (index, pointX, pointY, modifiers) => pane.itemMenuRequested(index, pointX, pointY, modifiers)
            onBackgroundMenuRequested: (pointX, pointY) => pane.backgroundMenuRequested(pointX, pointY)
            onRestoreRequested: pane.restoreRequested()
            onEmptyRequested: pane.emptyTrashRequested()
            onItemsDropped: drop => pane.dropOn(TrashService.filesPath, drop)
            onZoomRequested: (mode, level) => pane.setZoom(mode, level)
            onColumnResizeRequested: (id, width) => pane.setColumnWidth(id, width)
            onPreviewToggleRequested: pane.previewToggleRequested()
            onPreviewResized: width => pane.setPreviewWidth(width)
            onFocusRequested: pane.focusRequested()
            onBackRequested: pane.back()
            onForwardRequested: pane.forward()
        }
    }

    Component {
        id: paneStatus

        PaneStatus {
            total: browser.directory.total
            hidden: pane.showHidden ? 0 : browser.directory.hiddenCount
            compact: pane.width < Theme.smallBreakpoint
            showClipboard: pane.active
        }
    }

    Component {
        id: filterHeader

        Item {
            width: browser.body.width
            implicitHeight: filterRow.implicitHeight + Theme.spacingS * 2

            Row {
                id: filterRow

                anchors.left: parent.left
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingM

                StyledText {
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width - subfolderChip.width - parent.spacing
                    text: I18n.tr("%1 matching “%2”", "filter bar above a folder listing, count then the typed text").arg(FileFormat.count(browser.directory.count)).arg(pane.nameFilter)
                    color: Theme.surfaceVariantText
                    font.pixelSize: Theme.fontSizeSmall
                    elide: Text.ElideMiddle
                }

                DFilterChips {
                    id: subfolderChip

                    anchors.verticalCenter: parent.verticalCenter
                    width: implicitWidth
                    model: [
                        {
                            "value": "subfolders",
                            "label": I18n.tr("Search subfolders", "chip that turns the folder filter into a recursive search")
                        }
                    ]
                    multiSelect: true
                    selectedValues: []
                    showCounts: false
                    onSelectionToggled: pane.startSearch(pane.nameFilter, null)
                }
            }
        }
    }
}
