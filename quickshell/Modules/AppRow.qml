pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

DankListItem {
    id: row

    required property int index
    required property var modelData

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
            iconName: row.modelData.icon ?? ""
            size: Theme.iconSize
            color: row.contentColor
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 0

            StyledText {
                Layout.fillWidth: true
                text: row.modelData.name
                color: row.contentColor
                font.pixelSize: Theme.fontSizeMedium
                elide: Text.ElideRight
            }

            StyledText {
                Layout.fillWidth: true
                visible: text !== ""
                text: row.modelData.genericName ?? ""
                color: row.supportingContentColor
                font.pixelSize: Theme.fontSizeSmall
                elide: Text.ElideRight
            }
        }

        DankIcon {
            visible: row.modelData.isDefault === true
            name: "check"
            size: Theme.iconSizeMedium
            color: Theme.primary
        }
    }
}
