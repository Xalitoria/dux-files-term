pragma ComponentBehavior: Bound

import QtQuick
import qs.Common

// Placeholder rows for a directory that is still being listed, which only
// happens on a slow mount. The pulse runs on one animation for the whole
// column and stops with the loader that owns it.
Column {
    id: skeleton

    property real rowHeight: Theme.listItemHeight - Theme.spacingM
    property int rows: Math.max(1, Math.floor(height / (rowHeight + Theme.groupedListGap)))

    spacing: Theme.groupedListGap

    SequentialAnimation on opacity {
        running: Theme.animationsEnabled && skeleton.visible
        loops: Animation.Infinite

        NumberAnimation {
            to: 0.45
            duration: Theme.longDuration
            easing.type: Theme.standardEasing
        }

        NumberAnimation {
            to: 1
            duration: Theme.longDuration
            easing.type: Theme.standardEasing
        }
    }

    Repeater {
        model: skeleton.rows

        Rectangle {
            required property int index

            width: skeleton.width
            height: skeleton.rowHeight
            radius: Theme.groupedListInnerRadius
            color: Theme.surfaceContainerHigh

            Rectangle {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                width: Theme.iconSize
                height: Theme.iconSize
                radius: Theme.cornerRadiusXS
                color: Theme.surfaceContainerHighest
            }

            Rectangle {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingM * 2 + Theme.iconSize
                anchors.verticalCenter: parent.verticalCenter
                width: Math.min(parent.width / 2, Theme.spacingXL * (6 + index % 4))
                height: Theme.fontSizeMedium
                radius: Theme.cornerRadiusXS
                color: Theme.surfaceContainerHighest
            }
        }
    }
}
