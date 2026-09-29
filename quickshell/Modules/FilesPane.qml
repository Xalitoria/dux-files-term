pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

FocusScope {
    id: pane

    property bool active: true
    property bool showActiveOutline: false
    property bool showHidden: false
    property string sortKey: "name"
    property bool sortDescending: false
    property string viewMode: "list"
    property var cutSet: ({})
    property string nameFilter: ""

    readonly property alias path: browser.path
    readonly property alias canGoBack: browser.canGoBack
    readonly property alias canGoForward: browser.canGoForward
    readonly property alias canGoUp: browser.canGoUp
    readonly property alias editingPath: browser.editingPath
    readonly property alias renaming: browser.renaming
    readonly property alias searchModel: searchModel
    readonly property bool resultsActive: searchModel.active
    readonly property var selection: resultsActive ? results.selection : browser.selection
    readonly property var body: resultsActive ? results.body : browser.body
    readonly property var currentEntry: resultsActive ? results.body.cursorEntry() : browser.currentEntry
    readonly property var selectedEntries: resultsActive ? results.body.selectedEntries() : browser.selectedEntries
    readonly property bool isHome: path === Paths.strip(Paths.home)
    readonly property bool filtering: nameFilter !== "" && !resultsActive

    signal activateRequested(var entry)
    signal itemMenuRequested(int index, real pointX, real pointY, int modifiers)
    signal backgroundMenuRequested(real pointX, real pointY)
    signal dropRequested(var paths, string dest, string action)
    signal dragRequested(var paths)
    signal renameRequested(string target, string name)
    signal sortRequested(string key)
    signal columnsMenuRequested(real pointX, real pointY)
    signal focusRequested

    function navigate(target) {
        closeSearch();
        browser.navigate(target);
    }

    function reveal(target) {
        closeSearch();
        browser.reveal(target);
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
        closeSearch();
        browser.up();
    }

    function refresh() {
        if (resultsActive) {
            searchModel.refresh();
            return;
        }
        browser.refresh();
    }

    function editPath() {
        closeSearch();
        browser.editPath();
    }

    function focusBody() {
        if (resultsActive) {
            results.focusBody();
            return;
        }
        browser.focusBody();
    }

    function beginRename(target) {
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
        browser.restoreState(state);
    }

    function marker() {
        return browser.marker();
    }

    function startSearch(query, filterState) {
        if (path === "")
            return;
        const state = filterState ?? results.chips.filterState;
        searchModel.rootPath = path;
        searchModel.query = query;
        searchModel.options = SearchFilters.options(state);
        searchModel.start();
        results.focusBody();
    }

    function closeSearch() {
        if (!resultsActive)
            return;
        searchModel.clear();
        nameFilter = "";
    }

    function dropOn(dest, drop) {
        const paths = DragService.pathsFrom(drop);
        if (paths.length === 0 || paths.includes(dest))
            return;
        dropRequested(paths, dest, DragService.actionFor(drop.modifiers ?? Qt.NoModifier));
    }

    function setZoom(mode, level) {
        if (mode === "grid") {
            SessionData.gridZoom = level;
            return;
        }
        SessionData.listZoom = level;
    }

    function setColumnWidth(id, width) {
        const widths = Object.assign({}, SessionData.columnWidths);
        widths[id] = width;
        SessionData.columnWidths = widths;
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
        visible: !pane.resultsActive
        focus: !pane.resultsActive
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
        cutSet: pane.cutSet
        nameFilter: pane.nameFilter
        singleClickActivates: SettingsData.clickToOpen === "single"
        thumbnailsEnabled: SettingsData.thumbnails
        thumbnailSizeLimit: SettingsData.thumbnailSizeLimitMb > 0 ? SettingsData.thumbnailSizeLimitMb * 1000000 : -1
        networkThumbnails: SettingsData.networkThumbnails
        pathSuggestions: SessionData.recentDirs
        header: pane.filtering ? filterHeader : (pane.isHome ? quickAccess : null)
        emptyText: pane.filtering ? I18n.tr("No results", "search results header when nothing matched") : (pane.isHome ? "" : I18n.tr("This folder is empty", "empty directory placeholder"))
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
        onFocusRequested: pane.focusRequested()
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
        cutSet: pane.cutSet
        singleClickActivates: SettingsData.clickToOpen === "single"
        border.width: pane.showActiveOutline && pane.active ? Theme.outlineWidthFocused : 0
        border.color: Theme.primary
        onActivateRequested: entry => pane.activateRequested(entry)
        onItemMenuRequested: (index, pointX, pointY, modifiers) => pane.itemMenuRequested(index, pointX, pointY, modifiers)
        onDragRequested: paths => pane.dragRequested(paths)
        onRenameRequested: (target, name) => pane.renameRequested(target, name)
        onZoomRequested: (mode, level) => pane.setZoom(mode, level)
        onColumnResizeRequested: (id, width) => pane.setColumnWidth(id, width)
        onFocusRequested: pane.focusRequested()
        onCloseRequested: {
            pane.closeSearch();
            pane.focusBody();
        }
        onFiltersChanged: pane.startSearch(searchModel.query, results.chips.filterState)
    }

    Component {
        id: quickAccess

        QuickAccessBand {
            width: browser.body.width
            onOpened: target => {
                pane.focusRequested();
                pane.navigate(target);
            }
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

                DankFilterChips {
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
