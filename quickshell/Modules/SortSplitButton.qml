pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Row {
    id: control

    property string sortKey: "name"
    property bool descending: false

    signal cycleRequested
    signal directionToggled

    readonly property string sortIcon: {
        switch (sortKey) {
        case "mtime":
            return "history";
        case "size":
            return "straighten";
        case "type":
            return "category";
        default:
            return "sort_by_alpha";
        }
    }
    readonly property string sortLabel: {
        switch (sortKey) {
        case "mtime":
            return I18n.tr("Sort by date modified", "tooltip on the sort button");
        case "size":
            return I18n.tr("Sort by size", "tooltip on the sort button");
        case "type":
            return I18n.tr("Sort by type", "tooltip on the sort button");
        default:
            return I18n.tr("Sort by name", "tooltip on the sort button");
        }
    }

    spacing: FilesMetrics.navPairGap
    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    component Segment: StyledButton {
        id: segment

        property string iconName: ""
        property string tooltip: ""
        property bool round: false
        property bool leading: false

        readonly property real outerRadius: Theme.fullRadius(width, height)
        readonly property real startRadius: round || leading ? outerRadius : FilesMetrics.navPairInnerRadius
        readonly property real endRadius: round || !leading ? outerRadius : FilesMetrics.navPairInnerRadius

        width: FilesMetrics.topbarControlSize
        height: FilesMetrics.topbarControlSize
        color: Theme.surfaceContainerHigh
        topLeftRadius: mirrored ? endRadius : startRadius
        bottomLeftRadius: mirrored ? endRadius : startRadius
        topRightRadius: mirrored ? startRadius : endRadius
        bottomRightRadius: mirrored ? startRadius : endRadius
        Accessible.name: tooltip

        Behavior on topRightRadius {
            enabled: Theme.animationsEnabled
            NumberAnimation {
                duration: Theme.expressiveDurations.expressiveEffects
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Theme.expressiveCurves.expressiveEffects
            }
        }

        Behavior on bottomRightRadius {
            enabled: Theme.animationsEnabled
            NumberAnimation {
                duration: Theme.expressiveDurations.expressiveEffects
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Theme.expressiveCurves.expressiveEffects
            }
        }

        StateLayer {
            control: segment
            stateColor: Theme.surfaceText
            tooltipText: segment.tooltip
        }

        DankIcon {
            anchors.centerIn: parent
            name: segment.iconName
            size: Theme.iconSizeMedium
            color: Theme.surfaceText
        }

        FocusRing {
            visible: segment.visualFocus
            radius: segment.outerRadius
        }
    }

    Segment {
        leading: true
        iconName: control.sortIcon
        tooltip: control.sortLabel
        onClicked: control.cycleRequested()
    }

    Segment {
        round: control.descending
        iconName: control.descending ? "arrow_downward" : "arrow_upward"
        tooltip: control.descending ? I18n.tr("Sort descending", "tooltip on the sort direction button") : I18n.tr("Sort ascending", "tooltip on the sort direction button")
        onClicked: control.directionToggled()
    }
}
