pragma ComponentBehavior: Bound

import QtQuick
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

    DirectoryModel {
        id: directory
        path: window.currentPath
    }

    Item {
        anchors.fill: parent
        readonly property bool isFloatingWindowSurface: true

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
            anchors.top: pathLabel.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: Theme.spacingL
            clip: true
            spacing: Theme.groupedListGap
            model: directory.entries
            delegate: FileRow {
                onClicked: {
                    if (isDir)
                        window.currentPath = path;
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
