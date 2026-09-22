pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Models
import qs.Services
import qs.DankCommon.Widgets

FocusScope {
    id: body

    required property var directory
    required property SelectionModel selection
    required property string path

    property bool showHidden: false

    property string viewMode: SessionData.viewMode
    property bool isHome: path === Paths.strip(Paths.home)
    property string sortKey: "name"
    property bool sortDescending: false
    property string renamingPath: ""
    property var cutSet: ({})
    property var childCounts: ({})

    readonly property var view: viewLoader.item
    readonly property int zoom: viewMode === "grid" ? SessionData.gridZoom : SessionData.listZoom
    readonly property real iconSize: viewMode === "grid" ? FilesMetrics.gridIconSizes[zoom] : FilesMetrics.listIconSizes[zoom]
    readonly property bool listing: directory.loading && directory.count === 0
    readonly property bool empty: !directory.loading && directory.error === "" && directory.count === 0

    signal activateRequested(var entry)
    signal itemMenuRequested(int index, real pointX, real pointY, int modifiers)
    signal backgroundMenuRequested(real pointX, real pointY)
    signal columnsMenuRequested(real pointX, real pointY)
    signal dropRequested(var paths, string dest, string action)
    signal dragRequested(var paths)
    signal renameRequested(string path, string name)
    signal openRequested(string path)
    signal focusRequested
    signal sortRequested(string key)

    function focusBody() {
        forceActiveFocus();
    }

    function entryAt(index) {
        return index >= 0 && index < directory.count ? directory.entries.get(index) : null;
    }

    function cursorIndex() {
        return selection.cursorPath === "" ? -1 : selection.indexOf(directory.entries, selection.cursorPath);
    }

    function cursorEntry() {
        return entryAt(cursorIndex());
    }

    function selectedEntries() {
        return selection.paths.map(selected => {
            const index = selection.indexOf(directory.entries, selected);
            return entryAt(index);
        }).filter(entry => entry !== null);
    }

    function beginRename(target) {
        renamingPath = target;
    }

    function setZoom(level) {
        const clamped = Math.max(0, Math.min(FilesMetrics.gridIconSizes.length - 1, level));
        if (viewMode === "grid") {
            SessionData.gridZoom = clamped;
            return;
        }
        SessionData.listZoom = clamped;
    }

    function zoomBy(step) {
        setZoom(zoom + step);
    }

    function activate(index) {
        const entry = entryAt(index);
        if (!entry)
            return;
        selection.select(entry.path);
        activateRequested(entry);
    }

    function click(index, modifiers) {
        focusRequested();
        const entry = entryAt(index);
        if (!entry)
            return;
        if ((modifiers & Qt.ShiftModifier) !== 0) {
            selection.extendTo(directory.entries, index);
            return;
        }
        if ((modifiers & Qt.ControlModifier) !== 0) {
            selection.toggle(entry.path);
            return;
        }
        selection.select(entry.path);
    }

    function moveCursor(step, modifiers) {
        if (directory.count === 0)
            return;
        const current = cursorIndex();
        const next = Math.max(0, Math.min(directory.count - 1, current < 0 ? 0 : current + step));
        view?.positionViewAtIndex(next, ListView.Contain);
        if ((modifiers & Qt.ShiftModifier) !== 0) {
            selection.extendTo(directory.entries, next);
            return;
        }
        selection.select(directory.entries.get(next).path);
    }

    function pageStep() {
        return viewMode === "grid" ? Math.max(1, Math.floor(view.height / view.cellHeight)) * view.columnsPerRow : Math.max(1, Math.floor(view.height / view.rowPitch));
    }

    function jumpTo(index, modifiers) {
        if (directory.count === 0)
            return;
        const target = Math.max(0, Math.min(directory.count - 1, index));
        view?.positionViewAtIndex(target, ListView.Contain);
        if ((modifiers & Qt.ShiftModifier) !== 0) {
            selection.extendTo(directory.entries, target);
            return;
        }
        selection.select(directory.entries.get(target).path);
    }

    function typeAhead(text) {
        const prefix = (typeAheadTimer.prefix + text).toLowerCase();
        typeAheadTimer.prefix = prefix;
        typeAheadTimer.restart();
        for (let i = 0; i < directory.count; i++) {
            const entry = directory.entries.get(i);
            if (!entry.name.toLowerCase().startsWith(prefix))
                continue;
            jumpTo(i, Qt.NoModifier);
            return;
        }
    }

    // Directory sizes are item counts, asked for only over the rows on screen
    // that have no answer yet, on the same settle as the thumbnails.
    function requestCounts(first, last) {
        const wanted = [];
        for (let i = first; i <= Math.min(last, directory.count - 1); i++) {
            const entry = entryAt(i);
            if (!entry?.isDir || childCounts[entry.path] !== undefined)
                continue;
            wanted.push(entry.path);
        }
        if (wanted.length === 0)
            return;
        DirectoryService.count(wanted, showHidden, result => {
            if (result.error)
                return;
            const merged = Object.assign({}, body.childCounts);
            for (const path in result.counts || ({})) {
                const count = result.counts[path];
                merged[path] = count.code ? -1 : count.count;
            }
            body.childCounts = merged;
        });
    }

    function dropOn(dest, drop) {
        const paths = DragService.pathsFrom(drop);
        if (paths.length === 0 || paths.includes(dest))
            return;
        dropRequested(paths, dest, DragService.actionFor(drop.modifiers ?? Qt.NoModifier));
    }

    onPathChanged: {
        selection.clear();
        renamingPath = "";
        childCounts = ({});
    }

    onViewModeChanged: renamingPath = ""

    Connections {
        target: body.directory

        function onListed() {
            body.selection.prune(body.directory.entries);
            settle.restart();
        }
    }

    Timer {
        id: typeAheadTimer

        property string prefix: ""

        interval: 1000
        onTriggered: prefix = ""
    }

    Timer {
        id: settle

        interval: Theme.mediumDuration
        onTriggered: {
            if (!body.view)
                return;
            const range = body.view.visibleRange();
            body.directory.requestThumbnails(range[0], range[1]);
            body.requestCounts(range[0], range[1]);
            if (range[1] >= body.directory.count - 1)
                body.directory.loadMore();
        }
    }

    Keys.onPressed: event => {
        const modifiers = event.modifiers;
        const columns = view?.columnsPerRow ?? 1;
        switch (event.key) {
        case Qt.Key_Down:
            moveCursor(columns, modifiers);
            break;
        case Qt.Key_Up:
            moveCursor(-columns, modifiers);
            break;
        case Qt.Key_Right:
            moveCursor(body.viewMode === "grid" ? 1 : columns, modifiers);
            break;
        case Qt.Key_Left:
            moveCursor(body.viewMode === "grid" ? -1 : -columns, modifiers);
            break;
        case Qt.Key_Home:
            jumpTo(0, modifiers);
            break;
        case Qt.Key_End:
            jumpTo(directory.count - 1, modifiers);
            break;
        case Qt.Key_PageDown:
            moveCursor(pageStep(), modifiers);
            break;
        case Qt.Key_PageUp:
            moveCursor(-pageStep(), modifiers);
            break;
        case Qt.Key_Space:
            if (selection.cursorPath !== "")
                selection.toggle(selection.cursorPath);
            break;
        case Qt.Key_Escape:
            selection.clear();
            break;
        case Qt.Key_Backspace:
            if (typeAheadTimer.prefix !== "") {
                typeAheadTimer.prefix = "";
                break;
            }
            openRequested("..");
            break;
        default:
            if (modifiers !== Qt.NoModifier && modifiers !== Qt.ShiftModifier)
                return;
            if (event.text === "" || event.text.charCodeAt(0) < 0x20)
                return;
            typeAhead(event.text);
        }
        event.accepted = true;
    }

    WheelHandler {
        acceptedModifiers: Qt.ControlModifier
        onWheel: event => body.zoomBy(event.angleDelta.y > 0 ? 1 : -1)
    }

    Component {
        id: listComponent

        FileListView {
            id: listView

            selection: body.selection
            iconSize: body.iconSize
            renamingPath: body.renamingPath
            cutSet: body.cutSet
            childCounts: body.childCounts
            columnIds: SessionData.listColumns
            storedWidths: SessionData.columnWidths
            model: body.directory.entries
            header: bandComponent
            onItemClicked: (index, modifiers) => body.click(index, modifiers)
            onItemActivated: index => body.activate(index)
            onItemContextMenu: (index, pointX, pointY, modifiers) => body.itemMenuRequested(index, pointX, pointY, modifiers)
            onItemDragStarted: index => body.startDrag(index)
            onItemDropped: (index, drop) => body.dropOn(body.entryAt(index).path, drop)
            onRenameSubmitted: (target, name) => body.renameRequested(target, name)
            onRenameCancelled: body.renamingPath = ""
            onContentYChanged: settle.restart()
            onCountChanged: settle.restart()

            FileViewBackground {
                view: listView
                selection: body.selection
                onCleared: {
                    body.focusRequested();
                    body.selection.clear();
                }
                onContextMenuRequested: (pointX, pointY) => body.backgroundMenuRequested(pointX, pointY)
                onDropped: drop => body.dropOn(body.path, drop)
            }
        }
    }

    Component {
        id: gridComponent

        FileGridView {
            id: gridView

            selection: body.selection
            iconSize: body.iconSize
            renamingPath: body.renamingPath
            cutSet: body.cutSet
            model: body.directory.entries
            header: bandComponent
            onItemClicked: (index, modifiers) => body.click(index, modifiers)
            onItemActivated: index => body.activate(index)
            onItemContextMenu: (index, pointX, pointY, modifiers) => body.itemMenuRequested(index, pointX, pointY, modifiers)
            onItemDragStarted: index => body.startDrag(index)
            onItemDropped: (index, drop) => body.dropOn(body.entryAt(index).path, drop)
            onRenameSubmitted: (target, name) => body.renameRequested(target, name)
            onRenameCancelled: body.renamingPath = ""
            onContentYChanged: settle.restart()
            onCountChanged: settle.restart()

            FileViewBackground {
                view: gridView
                selection: body.selection
                onCleared: {
                    body.focusRequested();
                    body.selection.clear();
                }
                onContextMenuRequested: (pointX, pointY) => body.backgroundMenuRequested(pointX, pointY)
                onDropped: drop => body.dropOn(body.path, drop)
            }
        }
    }

    Component {
        id: bandComponent

        QuickAccessBand {
            width: body.width
            visible: body.isHome && dirs.length > 0
            height: visible ? implicitHeight + Theme.spacingM : 0
            onOpened: target => body.openRequested(target)
        }
    }

    FileColumnHeader {
        id: columnHeader

        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        visible: body.viewMode === "list"
        height: visible ? implicitHeight : 0
        columnIds: SessionData.listColumns
        widths: SessionData.columnWidths
        sortKey: body.sortKey
        sortDescending: body.sortDescending
        nameIndent: body.iconSize + Theme.spacingM * 2
        onSortRequested: key => body.sortRequested(key)
        onColumnsMenuRequested: (pointX, pointY) => body.columnsMenuRequested(pointX, pointY)
        onColumnResized: (id, width) => {
            const widths = Object.assign({}, SessionData.columnWidths);
            widths[id] = width;
            SessionData.columnWidths = widths;
        }
    }

    Loader {
        id: viewLoader

        anchors.top: columnHeader.bottom
        anchors.bottom: parent.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        sourceComponent: body.viewMode === "grid" ? gridComponent : listComponent
    }

    LoadingSkeleton {
        anchors.fill: viewLoader
        visible: body.listing
        rowHeight: body.viewMode === "grid" ? FilesMetrics.gridIconSizes[body.zoom] : body.iconSize + Theme.spacingM * 2
    }

    StyledText {
        anchors.centerIn: viewLoader
        visible: body.empty && !body.isHome
        text: I18n.tr("This folder is empty", "empty directory placeholder")
        color: Theme.surfaceVariantText
        font.pixelSize: Theme.fontSizeLarge
    }

    StyledText {
        anchors.centerIn: viewLoader
        width: viewLoader.width - Theme.spacingL * 2
        visible: body.directory.error !== ""
        text: body.errorText(body.directory.errorCode)
        color: Theme.surfaceVariantText
        font.pixelSize: Theme.fontSizeLarge
        horizontalAlignment: Text.AlignHCenter
        wrapMode: Text.Wrap
    }

    SelectionFooter {
        anchors.bottom: parent.bottom
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottomMargin: Theme.spacingM
        visible: body.selection.count > 0
        width: Math.min(implicitWidth, body.width - Theme.spacingL * 2)
        count: body.selection.count
        bytes: body.selectedBytes()
        onCleared: body.selection.clear()
    }

    function errorText(code) {
        switch (code) {
        case "ENOENT":
            return I18n.tr("This folder no longer exists", "directory listing error");
        case "EACCES":
            return I18n.tr("You do not have permission to view this folder", "directory listing error");
        default:
            return I18n.tr("This folder could not be opened", "directory listing error");
        }
    }

    function selectedBytes() {
        let total = 0;
        for (const entry of selectedEntries()) {
            if (entry.size < 0)
                return -1;
            total += entry.size;
        }
        return total;
    }

    function startDrag(index) {
        const entry = entryAt(index);
        if (!entry)
            return;
        if (!selection.contains(entry.path))
            selection.select(entry.path);
        dragRequested(selection.paths.slice());
    }
}
