pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Item {
    id: root

    property string iconName: ""
    property string thumbnail: ""
    property string path: ""
    property string mime: ""
    property bool isDir: false
    property bool isSymlink: false
    property bool symlinkBroken: false
    property bool untrusted: false
    property bool unreadable: false
    property bool hidden: false
    property bool cut: false
    property int iconSize: Theme.iconSize
    property color tint: Theme.surfaceText

    readonly property real emblemSize: Math.max(FilesMetrics.emblemSize, iconSize * 0.3)

    implicitWidth: iconSize
    implicitHeight: iconSize
    opacity: cut ? Theme.pendingOpacity : hidden ? 0.7 : 1

    FileIcon {
        id: glyph

        anchors.fill: parent
        iconName: root.iconName
        thumbnail: root.thumbnail !== "" ? root.thumbnail : root.mime === "image/svg+xml" ? root.path : ""
        size: root.iconSize
        color: root.isDir ? Theme.primary : root.tint
    }

    component Emblem: Rectangle {
        property alias iconName: badge.name
        property alias iconColor: badge.color

        width: root.emblemSize
        height: root.emblemSize
        radius: width / 2
        color: Theme.surfaceContainerLowest

        DankIcon {
            id: badge

            anchors.centerIn: parent
            size: parent.width * 0.8
        }
    }

    Emblem {
        anchors.left: parent.left
        anchors.bottom: parent.bottom
        visible: root.isSymlink
        iconName: root.symlinkBroken ? "link_off" : "link"
        iconColor: root.symlinkBroken ? Theme.error : Theme.surfaceVariantText
    }

    Emblem {
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        visible: root.untrusted || root.unreadable
        iconName: root.unreadable ? "lock" : "warning"
        iconColor: Theme.error
    }
}
