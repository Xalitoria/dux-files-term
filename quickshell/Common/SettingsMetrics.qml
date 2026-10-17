pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common

Singleton {
    readonly property real sidebarWidth: Theme.sidebarWidth + Theme.spacingXL
    readonly property real compactBreakpoint: Theme.smallBreakpoint + Theme.sidebarWidth / 2
    readonly property real contentMaxWidth: Theme.smallBreakpoint + Theme.spacingXL * 2
    readonly property real pagePaddingV: Theme.spacingXL
    readonly property real rowPaddingH: Theme.spacingL + Theme.spacingXS
    readonly property real rowPaddingV: Theme.spacingL
    readonly property real rowContentSpacing: Theme.spacingL
    readonly property real sectionLabelTopGap: Theme.spacingS
    readonly property real sectionLabelBottomGap: Theme.spacingM
    readonly property real navIconSize: Theme.avatarSize
    readonly property real navItemMinHeight: Theme.listItemHeight
    readonly property real sidebarGroupGap: Theme.spacingS
    readonly property real buttonGroupCompactThreshold: Theme.spacingXL * 8
    readonly property real dropdownWidth: Theme.spacingXL * 7
    readonly property real disabledOpacity: 0.38
    readonly property real highlightBlend: 0.2
    readonly property color rowColor: Theme.foregroundColor(Theme.cardSurface, true)
    readonly property color selectedRowColor: Theme.blend(rowColor, Theme.onSurface, Theme.stateLayerDrag)
    readonly property int transitionDuration: Theme.expressiveDurations.expressiveFastSpatial
    readonly property int fadeDuration: Theme.expressiveDurations.expressiveEffects
    readonly property int flashHold: Anims.durLong * 2
}
