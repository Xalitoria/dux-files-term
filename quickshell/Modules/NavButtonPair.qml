pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Row {
    id: pair

    property bool backEnabled: false
    property bool forwardEnabled: false

    signal backRequested
    signal forwardRequested

    spacing: FilesMetrics.navPairGap
    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    component Segment: StyledButton {
        id: segment

        required property bool leading
        property string iconName: ""
        property string tooltipText: ""
        readonly property real outerRadius: Theme.fullRadius(FilesMetrics.navPairSegmentWidth, FilesMetrics.topbarControlSize)
        readonly property real startRadius: leading ? outerRadius : FilesMetrics.navPairInnerRadius
        readonly property real endRadius: leading ? FilesMetrics.navPairInnerRadius : outerRadius

        width: FilesMetrics.navPairSegmentWidth
        height: FilesMetrics.topbarControlSize
        color: enabled ? Theme.surfaceContainerHigh : Theme.onSurface_12
        topLeftRadius: mirrored ? endRadius : startRadius
        bottomLeftRadius: mirrored ? endRadius : startRadius
        topRightRadius: mirrored ? startRadius : endRadius
        bottomRightRadius: mirrored ? startRadius : endRadius
        Accessible.name: tooltipText

        StateLayer {
            control: segment
            disabled: !segment.enabled
            stateColor: Theme.surfaceText
            tooltipText: segment.tooltipText
        }

        DankIcon {
            anchors.centerIn: parent
            name: segment.iconName
            size: Theme.iconSizeMedium
            color: segment.enabled ? Theme.surfaceText : Theme.onSurface_38
        }

        FocusRing {
            visible: segment.visualFocus
            radius: segment.outerRadius
        }
    }

    Segment {
        leading: true
        enabled: pair.backEnabled
        iconName: I18n.isRtl ? "arrow_forward" : "arrow_back"
        tooltipText: I18n.tr("Back", "tooltip on the button going to the previous location")
        onClicked: pair.backRequested()
    }

    Segment {
        leading: false
        enabled: pair.forwardEnabled
        iconName: I18n.isRtl ? "arrow_back" : "arrow_forward"
        tooltipText: I18n.tr("Forward", "tooltip on the button going to the next location")
        onClicked: pair.forwardRequested()
    }
}
