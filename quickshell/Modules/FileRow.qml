pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.DankCommon.Widgets

DankListItem {
    id: row

    required property int index
    required property string name
    required property string path
    required property bool isDir
    required property bool isSymlink
    required property real size
    required property real mtimeMs
    required property string iconName
    required property string thumbnail
    required property string mime

    width: listView?.width ?? 0
    isSelected: (listView?.currentIndex ?? -1) === index
    firstInGroup: index === 0
    lastInGroup: index === (listView?.count ?? 0) - 1

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingL
        anchors.rightMargin: Theme.spacingL
        spacing: Theme.spacingM

        FileIcon {
            iconName: row.iconName
            thumbnail: row.thumbnail !== "" ? row.thumbnail : row.mime === "image/svg+xml" ? row.path : ""
            size: Theme.iconSize
            color: row.isDir ? Theme.primary : row.contentColor
        }

        StyledText {
            Layout.fillWidth: true
            text: row.name
            color: row.contentColor
            font.pixelSize: Theme.fontSizeMedium
            elide: Text.ElideMiddle
        }

        DankIcon {
            visible: row.isSymlink
            name: "link"
            size: Theme.fontSizeSmall
            color: row.supportingContentColor
        }

        StyledText {
            visible: row.size >= 0
            text: FileFormat.size(row.size)
            color: row.supportingContentColor
            font.pixelSize: Theme.fontSizeSmall
        }

        StyledText {
            Layout.preferredWidth: Theme.spacingXL * 6
            horizontalAlignment: Text.AlignRight
            text: FileFormat.modified(row.mtimeMs)
            color: row.supportingContentColor
            font.pixelSize: Theme.fontSizeSmall
        }
    }
}
