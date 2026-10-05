pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

Rectangle {
    id: ghost

    property int count: 0
    property string iconName: ""
    property string thumbnail: ""
    property string mime: ""
    property string path: ""
    property bool isDir: false
    property string action: "move"

    readonly property string actionLabel: {
        switch (action) {
        case "copy":
            return I18n.tr("Copy", "context menu entry copying the selection");
        case "link":
            return I18n.tr("Link", "drag label for creating a symlink");
        default:
            return I18n.tr("Move", "drag label for moving files");
        }
    }

    implicitWidth: body.implicitWidth + Theme.spacingM * 2
    implicitHeight: FilesMetrics.dragGhostSize + Theme.spacingS * 2
    radius: Theme.cornerRadiusL
    color: Theme.surfaceContainerHigh
    border.width: Theme.outlineWidth
    border.color: Theme.primary

    Row {
        id: body

        anchors.centerIn: parent
        spacing: Theme.spacingS

        Item {
            anchors.verticalCenter: parent.verticalCenter
            width: FilesMetrics.dragGhostSize
            height: FilesMetrics.dragGhostSize

            FileIcon {
                anchors.fill: parent
                iconName: ghost.iconName
                thumbnail: ghost.thumbnail !== "" ? ghost.thumbnail : ghost.mime === "image/svg+xml" ? ghost.path : ""
                size: FilesMetrics.dragGhostSize
                color: ghost.isDir ? Theme.primary : Theme.surfaceText
            }

            Rectangle {
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                visible: ghost.count > 1
                width: Math.max(height, badge.implicitWidth + Theme.spacingXS)
                height: Theme.iconSizeSmall
                radius: Theme.fullRadius(width, height)
                color: Theme.primary

                StyledText {
                    id: badge

                    anchors.centerIn: parent
                    text: ghost.count
                    color: Theme.onPrimary
                    font.pixelSize: Theme.fontSizeSmall
                    font.weight: Theme.fontWeightMedium
                }
            }
        }

        StyledText {
            anchors.verticalCenter: parent.verticalCenter
            text: ghost.actionLabel
            color: Theme.surfaceText
            font.pixelSize: Theme.fontSizeSmall
            font.weight: Theme.fontWeightMedium
        }
    }
}
