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

    readonly property alias path: history.path
    readonly property alias canGoBack: history.canGoBack
    readonly property alias canGoForward: history.canGoForward
    readonly property alias canGoUp: history.canGoUp
    readonly property alias editingPath: pathBar.editing
    readonly property int currentIndex: fileList.currentIndex
    readonly property var currentEntry: fileList.currentIndex >= 0 && fileList.currentIndex < directory.count ? directory.entries.get(fileList.currentIndex) : null
    readonly property bool empty: !directory.loading && directory.error === "" && directory.count === 0

    signal activateRequested(var entry)
    signal contextMenuRequested(int index, real pointX, real pointY)
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
        fileList.forceActiveFocus();
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
            "contentY": fileList.contentY,
            "index": fileList.currentIndex
        };
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

    NavigationHistory {
        id: history

        onRestored: saved => {
            fileList.currentIndex = saved?.index ?? -1;
            restoreScroll.pending = saved?.contentY ?? 0;
            restoreScroll.restart();
        }
    }

    Timer {
        id: restoreScroll

        property real pending: 0

        interval: 0
        onTriggered: fileList.contentY = Math.max(0, Math.min(pending, Math.max(0, fileList.contentHeight - fileList.height)))
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

        DankListView {
            id: fileList

            anchors.top: pathBar.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: FilesMetrics.paneBodyMargin
            clip: true
            spacing: Theme.groupedListGap
            model: directory.entries
            cacheBuffer: Math.max(0, height)
            currentIndex: -1

            delegate: FileRow {
                onClicked: {
                    pane.focusRequested();
                    fileList.currentIndex = index;
                }

                onDoubleClicked: {
                    fileList.currentIndex = index;
                    pane.activateRequested(directory.entries.get(index));
                }

                onContextMenuRequested: (mouseX, mouseY) => {
                    pane.focusRequested();
                    const point = mapToItem(pane, mouseX, mouseY);
                    pane.contextMenuRequested(index, point.x, point.y);
                }
            }

            onContentYChanged: scrollSettle.restart()
            onCountChanged: scrollSettle.restart()

            Timer {
                id: scrollSettle

                interval: Theme.mediumDuration
                onTriggered: {
                    const first = fileList.indexAt(fileList.width / 2, fileList.contentY);
                    const last = fileList.indexAt(fileList.width / 2, fileList.contentY + fileList.height - 1);
                    directory.requestThumbnails(Math.max(first, 0), last < 0 ? fileList.count - 1 : last);
                    if (last < 0 || last >= fileList.count - 1)
                        directory.loadMore();
                }
            }
        }

        StyledText {
            anchors.centerIn: fileList
            visible: pane.empty
            text: I18n.tr("This folder is empty", "empty directory placeholder")
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeLarge
        }

        StyledText {
            anchors.centerIn: fileList
            width: fileList.width - FilesMetrics.paneBodyMargin * 2
            visible: directory.error !== ""
            text: pane.errorText(directory.errorCode)
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeLarge
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
        }

        TapHandler {
            onTapped: pane.focusRequested()
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
