pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Item {
    id: row

    property var tabs: []
    property int currentIndex: 0

    signal tabSelected(int index)
    signal tabClosed(int index)
    signal tabAdded

    implicitHeight: FilesMetrics.tabHeight
    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    DankFlickable {
        id: strip

        anchors.left: parent.left
        anchors.right: addButton.left
        anchors.rightMargin: FilesMetrics.tabSpacing
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        wheelEnabled: false
        clip: true
        flickableDirection: Flickable.HorizontalFlick
        contentWidth: strapline.width
        contentHeight: height

        WheelHandler {
            acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
            onWheel: event => {
                const delta = event.angleDelta.y !== 0 ? event.angleDelta.y : event.angleDelta.x;
                strip.contentX = Math.max(0, Math.min(Math.max(0, strip.contentWidth - strip.width), strip.contentX - delta));
            }
        }

        Row {
            id: strapline

            height: strip.height
            spacing: FilesMetrics.tabSpacing

            Repeater {
                model: row.tabs

                FilesTab {
                    required property int index
                    required property var modelData

                    label: modelData
                    current: index === row.currentIndex
                    closable: row.tabs.length > 1
                    onClicked: row.tabSelected(index)
                    onCloseRequested: row.tabClosed(index)
                }
            }
        }
    }

    DankActionButton {
        id: addButton

        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        buttonSize: Theme.buttonHeightXS
        iconName: "add"
        iconSize: Theme.iconSizeMedium
        iconColor: Theme.onSurfaceVariant
        tooltipText: I18n.tr("New tab", "tooltip on the button opening a new tab")
        onClicked: row.tabAdded()
    }

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: Theme.dividerWidth
        color: Theme.outlineVariant
    }
}
