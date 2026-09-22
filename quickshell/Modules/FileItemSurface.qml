pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

// The click, drag and drop behaviour every file delegate shares. Grid and list
// delegates fill it with their own layout and nothing else.
Rectangle {
    id: surface

    required property int index
    required property string path
    required property bool isDir

    property Item view: null
    property bool selected: false
    property bool cursor: false
    property bool cut: false
    property bool renaming: false
    property real contentRadius: Theme.groupedListInnerRadius

    readonly property bool dropTarget: isDir && !renaming && dropZone.containsDrag
    readonly property color contentColor: selected ? Theme.onSelectedContainer : Theme.surfaceText
    readonly property color supportingColor: selected ? Theme.onSelectedContainer : Theme.surfaceVariantText

    default property alias content: holder.data

    radius: contentRadius
    color: selected ? Theme.selectedContainer : dropTarget ? Theme.primaryContainer : "transparent"

    // Drawn inside the item, not at the usual negative offset: the views clip,
    // so an outset ring loses its left and right strokes at the viewport edge.
    Rectangle {
        anchors.fill: parent
        anchors.margins: Theme.focusRingOffset
        visible: surface.cursor && !surface.renaming
        radius: Math.max(0, parent.radius - Theme.focusRingOffset)
        color: "transparent"
        border.width: Theme.focusRingWidth
        border.color: Theme.focusRingColor
    }

    StateLayer {
        id: pointer

        acceptedButtons: Qt.LeftButton | Qt.RightButton
        enabled: !surface.renaming
        stateColor: surface.contentColor
        cornerRadius: surface.radius

        onClicked: mouse => {
            if (mouse.button === Qt.RightButton) {
                const point = mapToItem(surface.view, mouse.x, mouse.y);
                surface.view.itemContextMenu(surface.index, point.x, point.y, mouse.modifiers);
                return;
            }
            surface.view.itemClicked(surface.index, mouse.modifiers);
        }

        onDoubleClicked: mouse => {
            if (mouse.button !== Qt.LeftButton)
                return;
            surface.view.itemActivated(surface.index);
        }
    }

    DragHandler {
        target: null
        enabled: !surface.renaming
        onActiveChanged: {
            if (!active)
                return;
            surface.view.itemDragStarted(surface.index);
        }
    }

    DropArea {
        id: dropZone

        anchors.fill: parent
        enabled: surface.isDir && !surface.renaming
        onDropped: drop => surface.view.itemDropped(surface.index, drop)
    }

    Item {
        id: holder

        anchors.fill: parent
    }
}
