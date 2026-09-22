pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Models
import qs.Services

// Lives inside the view's contentItem behind the delegates, so a press that
// misses every item lands here: that is the rubber band, the background menu
// and the drop target for the directory itself.
Item {
    id: background

    required property Item view
    required property SelectionModel selection

    signal cleared
    signal contextMenuRequested(real pointX, real pointY)
    signal dropped(var drop)

    parent: view.contentItem
    z: -1
    width: view.width
    height: Math.max(view.contentHeight, view.height)

    MouseArea {
        id: area

        property var kept: []
        property bool banding: false

        anchors.fill: parent
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        preventStealing: true

        onPressed: mouse => {
            if (mouse.button === Qt.RightButton)
                return;
            kept = (mouse.modifiers & Qt.ControlModifier) !== 0 ? background.selection.paths.slice() : [];
            if (kept.length === 0)
                background.cleared();
            band.originX = mouse.x;
            band.originY = mouse.y;
            band.pointX = mouse.x;
            band.pointY = mouse.y;
            banding = true;
        }

        onPositionChanged: mouse => {
            if (!banding)
                return;
            band.pointX = mouse.x;
            band.pointY = mouse.y;
            if (!band.past)
                return;
            background.selection.paths = kept.concat(background.pathsUnder(Qt.rect(band.x, band.y, band.width, band.height)).filter(path => !kept.includes(path)));
        }

        onReleased: banding = false
        onCanceled: banding = false

        onClicked: mouse => {
            if (mouse.button !== Qt.RightButton)
                return;
            const point = mapToItem(background.view, mouse.x, mouse.y);
            background.contextMenuRequested(point.x, point.y);
        }
    }

    function pathsUnder(rect) {
        const model = view.model;
        return view.indicesIn(rect).map(index => model.get(index).path);
    }

    RubberBand {
        id: band

        visible: area.banding && past
    }

    DropArea {
        anchors.fill: parent
        onDropped: drop => background.dropped(drop)
    }
}
