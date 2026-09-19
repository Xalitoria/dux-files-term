pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Window
import Quickshell
import qs.Common
import qs.Models
import qs.DankCommon.Widgets

FloatingWindow {
    id: window

    signal hideRequested

    property string currentPath: Paths.strip(Paths.home)

    title: I18n.tr("Dank Files", "main window title")
    implicitWidth: 960
    implicitHeight: 640
    minimumSize: Qt.size(480, 320)
    color: Theme.surface
    visible: true

    onClosed: hideRequested()

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

    function navigateUp() {
        const parent = currentPath.substring(0, currentPath.lastIndexOf("/"));
        currentPath = parent === "" ? "/" : parent;
    }

    DirectoryModel {
        id: directory
        path: window.currentPath
        onGone: window.navigateUp()
    }

    Item {
        id: surface

        anchors.fill: parent
        readonly property bool isFloatingWindowSurface: true
        readonly property bool windowActive: Window.active

        onWindowActiveChanged: {
            if (windowActive)
                directory.refresh();
        }

        Shortcut {
            sequences: ["Ctrl+H"]
            onActivated: directory.setShowHidden(!directory.includeHidden)
        }

        StyledText {
            id: pathLabel
            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: Theme.spacingL
            text: Paths.shortenHome(window.currentPath)
            color: Theme.surfaceText
            font.pixelSize: Theme.fontSizeLarge
            elide: Text.ElideMiddle
        }

        DankListView {
            id: fileList

            anchors.top: pathLabel.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: Theme.spacingL
            clip: true
            spacing: Theme.groupedListGap
            model: directory.entries
            cacheBuffer: height

            delegate: FileRow {
                onClicked: {
                    if (isDir)
                        window.currentPath = path;
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
            anchors.centerIn: parent
            visible: !directory.loading && directory.error === "" && directory.count === 0
            text: I18n.tr("This folder is empty", "empty directory placeholder")
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeLarge
        }

        StyledText {
            anchors.centerIn: parent
            visible: directory.error !== ""
            text: window.errorText(directory.errorCode)
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeLarge
        }
    }
}
