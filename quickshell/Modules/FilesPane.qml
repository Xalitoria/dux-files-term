pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.FileBrowser

FocusScope {
    id: pane

    property bool active: true
    property bool showActiveOutline: false
    property bool showHidden: false
    property string sortKey: "name"
    property bool sortDescending: false
    property var cutSet: ({})

    readonly property alias path: browser.path
    readonly property alias canGoBack: browser.canGoBack
    readonly property alias canGoForward: browser.canGoForward
    readonly property alias canGoUp: browser.canGoUp
    readonly property alias editingPath: browser.editingPath
    readonly property alias renaming: browser.renaming
    readonly property alias selection: browser.selection
    readonly property alias body: browser.body
    readonly property alias currentEntry: browser.currentEntry
    readonly property alias selectedEntries: browser.selectedEntries
    readonly property bool isHome: path === Paths.strip(Paths.home)

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
        browser.navigate(target);
    }

    function back() {
        browser.back();
    }

    function forward() {
        browser.forward();
    }

    function up() {
        browser.up();
    }

    function refresh() {
        browser.refresh();
    }

    function editPath() {
        browser.editPath();
    }

    function focusBody() {
        browser.focusBody();
    }

    function beginRename(target) {
        browser.beginRename(target);
    }

    function saveState() {
        return browser.saveState();
    }

    function restoreState(state) {
        browser.restoreState(state);
    }

    function marker() {
        return browser.marker();
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

    FileBrowserView {
        id: browser

        anchors.fill: parent
        focus: true
        showSidebar: false
        showNavigation: false
        showHidden: pane.showHidden
        sortKey: pane.sortKey
        sortDescending: pane.sortDescending
        viewMode: SessionData.viewMode
        gridZoom: SessionData.gridZoom
        listZoom: SessionData.listZoom
        listColumns: SessionData.listColumns
        columnWidths: SessionData.columnWidths
        dragEnabled: true
        dropEnabled: true
        showSelectionFooter: true
        cutSet: pane.cutSet
        header: pane.isHome ? quickAccess : null
        emptyText: pane.isHome ? "" : I18n.tr("This folder is empty", "empty directory placeholder")
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
}
