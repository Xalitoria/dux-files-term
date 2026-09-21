pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

StyledButton {
    id: pill

    property string label: ""
    property string iconName: ""
    property bool current: false

    readonly property color contentColor: current ? Theme.onPrimaryContainer : Theme.surfaceText

    implicitWidth: row.implicitWidth + FilesMetrics.pathPillPadding * 2
    implicitHeight: FilesMetrics.pathPillHeight
    radius: FilesMetrics.pathPillRadius
    color: current ? Theme.primaryContainer : Theme.surfaceContainerHigh
    Accessible.name: label

    StateLayer {
        control: pill
        stateColor: pill.contentColor
    }

    FocusRing {
        visible: pill.visualFocus
    }

    Row {
        id: row

        anchors.centerIn: parent
        spacing: FilesMetrics.pathPillSpacing

        DankIcon {
            anchors.verticalCenter: parent.verticalCenter
            visible: pill.iconName !== ""
            name: pill.iconName
            size: FilesMetrics.pathIconSize
            color: pill.current ? pill.contentColor : Theme.primary
        }

        StyledText {
            anchors.verticalCenter: parent.verticalCenter
            visible: pill.label !== ""
            text: pill.label
            color: pill.contentColor
            font.pixelSize: Theme.fontSizeSmall
            font.weight: Font.Medium
        }
    }
}
