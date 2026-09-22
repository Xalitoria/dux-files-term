pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.DankCommon.Widgets

StyledButton {
    id: row

    property string label: ""
    property string iconName: ""
    property string actionIcon: ""
    property string actionTooltip: ""
    property real ringValue: -1
    property bool busy: false
    property bool selected: false

    property string dropPath: ""

    signal actionTriggered
    signal dropped(var drop)

    readonly property bool showAction: actionIcon !== "" && (hovered || actionArea.containsMouse)
    readonly property color contentColor: {
        if (!enabled)
            return Theme.onSurface_38;
        return selected ? Theme.onSecondaryContainer : Theme.surfaceText;
    }

    width: parent?.width ?? 0
    implicitHeight: FilesMetrics.sidebarRowHeight
    radius: Theme.cornerRadiusM
    color: dropZone.containsDrag ? Theme.primaryContainer : selected ? Theme.secondaryContainer : "transparent"

    DropArea {
        id: dropZone

        anchors.fill: parent
        enabled: row.dropPath !== ""
        onDropped: drop => row.dropped(drop)
    }

    StateLayer {
        control: row
        stateColor: row.contentColor
    }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingS
        anchors.rightMargin: Theme.spacingM
        spacing: Theme.spacingS

        DankRingGauge {
            id: gauge

            Layout.preferredWidth: FilesMetrics.sidebarIconSize
            Layout.preferredHeight: FilesMetrics.sidebarIconSize
            value: row.ringValue
            strokeWidth: Theme.spacingXXS
            ringColor: row.ringValue >= 0.9 ? Theme.error : Theme.primary
            trackColor: Theme.withAlpha(Theme.surfaceText, Theme.stateLayerHover)

            Rectangle {
                anchors.fill: parent
                radius: width / 2
                color: row.showAction ? Theme.primaryContainer : Theme.withAlpha(Theme.primary, Theme.stateLayerHover)

                DankSpinner {
                    anchors.centerIn: parent
                    visible: row.busy
                    size: Theme.iconSizeSmall
                    color: Theme.primary
                }

                FileIcon {
                    anchors.centerIn: parent
                    visible: !row.busy && !row.showAction
                    iconName: row.iconName
                    size: Theme.iconSizeSmall
                    color: Theme.primary
                }

                DankIcon {
                    anchors.centerIn: parent
                    visible: !row.busy && row.showAction
                    name: row.actionIcon
                    size: Theme.iconSizeSmall
                    color: Theme.onPrimaryContainer
                }

                MouseArea {
                    id: actionArea

                    anchors.fill: parent
                    enabled: row.actionIcon !== "" && !row.busy
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: row.actionTriggered()
                }
            }
        }

        StyledText {
            Layout.fillWidth: true
            text: row.label
            color: row.contentColor
            font.pixelSize: Theme.fontSizeSmall
            elide: Text.ElideMiddle
        }
    }
}
