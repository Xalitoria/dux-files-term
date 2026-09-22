pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    readonly property real sidebarWidth: Theme.sidebarWidth
    readonly property real sidebarPadding: Theme.spacingM
    readonly property real sidebarIconSize: Theme.iconSizeLarge
    readonly property real sidebarRowHeight: Theme.listItemHeight - Theme.spacingM
    readonly property real sidebarSectionSpacing: Theme.spacingL

    readonly property real topbarControlSize: Theme.buttonHeightS + Theme.spacingXS
    readonly property real topbarSpacing: Theme.spacingS
    readonly property real topbarGroupSpacing: Theme.spacingXS
    readonly property real navPairSegmentWidth: topbarControlSize + Theme.spacingS
    readonly property real navPairInnerRadius: Theme.cornerRadiusXS
    readonly property real navPairGap: Theme.spacingXXS

    readonly property real contentMargin: Theme.spacingL
    readonly property real contentSpacing: Theme.spacingM
    readonly property real paneGap: Theme.spacingM
    readonly property real paneRadius: Theme.cornerRadiusXL
    readonly property real paneBodyMargin: Theme.spacingM
    readonly property real splitterWidth: Theme.spacingM
    readonly property real splitterHandleHeight: Theme.iconSizeLarge
    readonly property real minimumPaneFraction: 0.2

    readonly property real pathBarSpacing: Theme.spacingXS
    readonly property real pathPillHeight: Theme.buttonHeightXXS
    readonly property real pathPillRadius: Theme.cornerRadiusS
    readonly property real pathPillPadding: Theme.spacingS
    readonly property real pathPillSpacing: Theme.spacingXS
    readonly property real pathIconSize: Theme.iconSizeSmall

    readonly property real tabHeight: Theme.buttonHeightS
    readonly property real tabMinWidth: Theme.buttonMinWidth
    readonly property real tabMaxWidth: Theme.sidebarWidth
    readonly property real tabSpacing: Theme.spacingXXS

    readonly property var gridIconSizes: [64, 96, 128, 192]
    readonly property var listIconSizes: [Theme.iconSizeMedium, Theme.iconSize, Theme.iconSizeLarge, Theme.iconSizeLarge * 1.5]
    readonly property real gridTilePadding: Theme.spacingM
    readonly property real gridTileRadius: Theme.cornerRadiusL
    readonly property real gridGap: Theme.spacingS
    readonly property real gridNameSpacing: Theme.spacingXS

    readonly property real columnHeaderHeight: Theme.buttonHeightXS
    readonly property real columnGap: Theme.spacingM
    readonly property real columnMinWidth: Theme.spacingXL * 3
    readonly property real sizeColumnWidth: Theme.spacingXL * 4
    readonly property real modifiedColumnWidth: Theme.spacingXL * 6
    readonly property real typeColumnWidth: Theme.spacingXL * 5
    readonly property real ownerColumnWidth: Theme.spacingXL * 4
    readonly property real permissionsColumnWidth: Theme.spacingXL * 4

    readonly property real selectionFooterHeight: Theme.buttonHeightXS + Theme.spacingS
    readonly property real emblemSize: Theme.fontSizeSmall
    readonly property real quickAccessTileSize: 104
    readonly property real dragGhostSize: Theme.iconSizeLarge * 2
    readonly property real rubberBandMinimum: Theme.spacingS
    readonly property int tabHoverDelay: 700

    readonly property real menuWidth: Theme.iconSize * 9
    readonly property int sidebarOverlayBreakpoint: Theme.mediumBreakpoint
}
