pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.DankCommon.FileBrowser

Singleton {
    id: root

    readonly property real windowMinWidth: Theme.smallBreakpoint
    readonly property real windowMinHeight: Theme.smallBreakpoint * 2 / 3
    readonly property real sidebarWidth: FileBrowserMetrics.sidebarWidth
    readonly property real sidebarPadding: FileBrowserMetrics.sidebarPadding
    readonly property real sidebarSectionSpacing: Theme.spacingL
    readonly property int sidebarOverlayBreakpoint: FileBrowserMetrics.sidebarBreakpoint

    readonly property real topbarControlSize: FileBrowserMetrics.controlSize
    readonly property real topbarSpacing: FileBrowserMetrics.controlSpacing
    readonly property real topbarGroupSpacing: Theme.spacingXS

    readonly property real contentMarginH: Theme.spacingXL
    readonly property real contentMarginV: FileBrowserMetrics.contentMargin
    readonly property real contentSpacing: FileBrowserMetrics.contentSpacing
    readonly property real paneRadius: FileBrowserMetrics.paneRadius
    readonly property real paneBodyMargin: FileBrowserMetrics.paneMargin
    readonly property real paneBodyMarginH: FileBrowserMetrics.paneMarginHorizontal
    readonly property real splitterWidth: Theme.spacingM
    readonly property real splitterHandleHeight: Theme.iconSizeLarge
    readonly property real minimumPaneFraction: 0.2

    readonly property real tabHeight: Theme.buttonHeightS
    readonly property int tabShortcutCount: 9
    readonly property int closedTabHistory: 10
    readonly property int defaultGridZoom: FileBrowserMetrics.defaultGridZoom
    readonly property int defaultListZoom: FileBrowserMetrics.defaultListZoom

    function defaultZoom(viewMode) {
        return FileBrowserMetrics.defaultZoom(viewMode);
    }
    readonly property real tabMinWidth: Theme.tabMinWidth
    readonly property real tabMaxWidth: Theme.sidebarWidth
    readonly property real tabSpacing: Theme.spacingXXS
    readonly property int tabHoverDelay: 700

    readonly property real dragGhostSize: FileBrowserMetrics.gridIconSizes[0]

    readonly property real popoverWidth: Theme.smallBreakpoint * 0.75
    readonly property real propertiesIconSize: FileBrowserMetrics.listIconSizes[3]
    readonly property real propertiesLabelWidth: Theme.spacingXL * 5
}
