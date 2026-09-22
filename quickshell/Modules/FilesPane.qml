pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Models
import qs.DankCommon.Widgets

FocusScope {
    id: pane

    property bool active: true
    property bool showActiveOutline: false
    property bool showHidden: false
    property string sortKey: "name"
    property bool sortDescending: false
    property var cutSet: ({})

    readonly property alias path: history.path
    readonly property alias canGoBack: history.canGoBack
    readonly property alias canGoForward: history.canGoForward
    readonly property alias canGoUp: history.canGoUp
    readonly property alias editingPath: pathBar.editing
    readonly property bool renaming: viewBody.renamingPath !== ""
    readonly property alias selection: selectionModel
    readonly property alias body: viewBody
    readonly property var currentEntry: viewBody.cursorEntry()
    readonly property var selectedEntries: viewBody.selectedEntries()

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
        _stamp();
        history.go(target);
    }

    function back() {
        _stamp();
        history.back();
    }

    function forward() {
        _stamp();
        history.forward();
    }

    function up() {
        _stamp();
        history.up();
    }

    function _stamp() {
        history.marker = marker();
    }

    function refresh() {
        directory.refresh();
    }

    function editPath() {
        pathBar.startEditing();
    }

    function focusBody() {
        viewBody.focusBody();
    }

    function beginRename(target) {
        viewBody.beginRename(target);
    }

    function saveState() {
        _stamp();
        return history.serialize();
    }

    function restoreState(state) {
        history.restore(state);
    }

    function marker() {
        return {
            "contentY": viewBody.view?.contentY ?? 0,
            "cursor": selectionModel.cursorPath
        };
    }

    onShowHiddenChanged: directory.setShowHidden(showHidden)
    onSortKeyChanged: directory.setSort(sortKey, sortDescending)
    onSortDescendingChanged: directory.setSort(sortKey, sortDescending)

    DirectoryModel {
        id: directory

        path: pane.path
        includeHidden: pane.showHidden
        sortKey: pane.sortKey
        sortDesc: pane.sortDescending
        onGone: history.up()
    }

    SelectionModel {
        id: selectionModel
    }

    NavigationHistory {
        id: history

        onRestored: saved => {
            if (saved?.cursor)
                selectionModel.select(saved.cursor);
            restoreScroll.pending = saved?.contentY ?? 0;
            restoreScroll.restart();
        }
    }

    Timer {
        id: restoreScroll

        property real pending: 0

        interval: 0
        onTriggered: {
            const view = viewBody.view;
            if (!view)
                return;
            view.contentY = Math.max(0, Math.min(pending, Math.max(0, view.contentHeight - view.height)));
        }
    }

    Rectangle {
        anchors.fill: parent
        radius: FilesMetrics.paneRadius
        color: Theme.surfaceContainerLow
        border.width: pane.showActiveOutline && pane.active ? Theme.outlineWidthFocused : 0
        border.color: Theme.primary

        PathBar {
            id: pathBar

            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: FilesMetrics.paneBodyMargin
            path: pane.path
            onNavigated: target => {
                pane.focusRequested();
                pane.navigate(target);
            }
        }

        FileViewBody {
            id: viewBody

            anchors.top: pathBar.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: FilesMetrics.paneBodyMargin
            directory: directory
            selection: selectionModel
            path: pane.path
            showHidden: pane.showHidden
            sortKey: pane.sortKey
            sortDescending: pane.sortDescending
            cutSet: pane.cutSet
            onActivateRequested: entry => pane.activateRequested(entry)
            onItemMenuRequested: (index, pointX, pointY, modifiers) => {
                pane.focusRequested();
                const point = mapToItem(pane, pointX, pointY);
                pane.itemMenuRequested(index, point.x, point.y, modifiers);
            }
            onBackgroundMenuRequested: (pointX, pointY) => {
                pane.focusRequested();
                const point = mapToItem(pane, pointX, pointY);
                pane.backgroundMenuRequested(point.x, point.y);
            }
            onColumnsMenuRequested: (pointX, pointY) => {
                const point = mapToItem(pane, pointX, pointY);
                pane.columnsMenuRequested(point.x, point.y);
            }
            onDropRequested: (paths, dest, action) => pane.dropRequested(paths, dest, action)
            onDragRequested: paths => pane.dragRequested(paths)
            onRenameRequested: (target, name) => pane.renameRequested(target, name)
            onSortRequested: key => pane.sortRequested(key)
            onFocusRequested: pane.focusRequested()
            onOpenRequested: target => {
                pane.focusRequested();
                if (target === "..") {
                    pane.up();
                    return;
                }
                pane.navigate(target);
            }
        }

        MouseArea {
            anchors.fill: parent
            acceptedButtons: Qt.BackButton | Qt.ForwardButton
            onClicked: mouse => {
                pane.focusRequested();
                if (mouse.button === Qt.BackButton) {
                    pane.back();
                    return;
                }
                pane.forward();
            }
        }
    }
}
