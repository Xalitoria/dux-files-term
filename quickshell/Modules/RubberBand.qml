pragma ComponentBehavior: Bound

import QtQuick
import qs.Common

Rectangle {
    id: band

    property real originX: 0
    property real originY: 0
    property real pointX: 0
    property real pointY: 0

    readonly property bool past: Math.abs(pointX - originX) > FilesMetrics.rubberBandMinimum || Math.abs(pointY - originY) > FilesMetrics.rubberBandMinimum

    x: Math.min(originX, pointX)
    y: Math.min(originY, pointY)
    width: Math.abs(pointX - originX)
    height: Math.abs(pointY - originY)
    radius: Theme.cornerRadiusXS
    color: Theme.withAlpha(Theme.primary, Theme.stateLayerHover)
    border.width: Theme.outlineWidth
    border.color: Theme.primary
}
