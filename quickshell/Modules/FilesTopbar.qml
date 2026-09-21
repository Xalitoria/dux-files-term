pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

Item {
    id: topbar

    property bool splitView: false
    property bool bookmarked: false
    property bool backEnabled: false
    property bool forwardEnabled: false
    property string sortKey: "name"
    property bool sortDescending: false

    readonly property alias searchText: search.text

    signal sidebarToggled
    signal backRequested
    signal forwardRequested
    signal splitToggled
    signal bookmarkToggled
    signal sortCycled
    signal sortDirectionToggled
    signal propertiesRequested
    signal copyPathRequested
    signal reloadRequested
    signal terminalRequested

    function focusSearch() {
        search.forceActiveFocus();
        search.selectAll();
    }

    implicitHeight: FilesMetrics.topbarControlSize
    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    component FlatButton: DankActionButton {
        buttonSize: FilesMetrics.topbarControlSize
        iconSize: Theme.iconSizeMedium
        iconColor: Theme.surfaceText
    }

    Row {
        id: leftActions

        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        spacing: FilesMetrics.topbarGroupSpacing

        FlatButton {
            anchors.verticalCenter: parent.verticalCenter
            iconName: "menu"
            tooltipText: I18n.tr("Toggle sidebar", "tooltip on the button showing or hiding the sidebar")
            onClicked: topbar.sidebarToggled()
        }

        NavButtonPair {
            anchors.verticalCenter: parent.verticalCenter
            backEnabled: topbar.backEnabled
            forwardEnabled: topbar.forwardEnabled
            onBackRequested: topbar.backRequested()
            onForwardRequested: topbar.forwardRequested()
        }
    }

    Row {
        id: rightActions

        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        spacing: FilesMetrics.topbarGroupSpacing

        FlatButton {
            anchors.verticalCenter: parent.verticalCenter
            iconName: "bookmark"
            iconFilled: topbar.bookmarked
            iconColor: topbar.bookmarked ? Theme.primary : Theme.surfaceText
            backgroundColor: Theme.surfaceContainerHigh
            tooltipText: topbar.bookmarked ? I18n.tr("Remove bookmark", "tooltip on the bookmark button") : I18n.tr("Bookmark this folder", "tooltip on the bookmark button")
            onClicked: topbar.bookmarkToggled()
        }

        FlatButton {
            anchors.verticalCenter: parent.verticalCenter
            iconName: "splitscreen_vertical_add"
            iconColor: topbar.splitView ? Theme.onPrimary : Theme.surfaceText
            backgroundColor: topbar.splitView ? Theme.primary : Theme.surfaceContainerHigh
            tooltipText: I18n.tr("Split view", "tooltip on the button showing a second pane")
            onClicked: topbar.splitToggled()
        }

        SortSplitButton {
            anchors.verticalCenter: parent.verticalCenter
            sortKey: topbar.sortKey
            descending: topbar.sortDescending
            onCycleRequested: topbar.sortCycled()
            onDirectionToggled: topbar.sortDirectionToggled()
        }

        FlatButton {
            id: overflowButton

            anchors.verticalCenter: parent.verticalCenter
            iconName: "more_vert"
            backgroundColor: Theme.surfaceContainerHigh
            tooltipText: I18n.tr("More actions", "tooltip on the overflow menu button")
            onClicked: overflowMenu.visible ? overflowMenu.close() : overflowMenu.open()

            ContextMenu {
                id: overflowMenu

                y: overflowButton.height + Theme.spacingXS
                x: overflowButton.width - width

                ContextMenuItem {
                    text: I18n.tr("Properties", "menu entry showing details about the current folder")
                    iconName: "info"
                    onClicked: {
                        overflowMenu.close();
                        topbar.propertiesRequested();
                    }
                }

                ContextMenuItem {
                    text: I18n.tr("Copy location path", "menu entry copying the current path to the clipboard")
                    iconName: "content_copy"
                    onClicked: {
                        overflowMenu.close();
                        topbar.copyPathRequested();
                    }
                }

                ContextMenuItem {
                    text: I18n.tr("Reload", "menu entry re-reading the current folder")
                    iconName: "refresh"
                    onClicked: {
                        overflowMenu.close();
                        topbar.reloadRequested();
                    }
                }

                ContextMenuItem {
                    visible: SandboxService.available("terminal")
                    text: I18n.tr("Open in terminal", "context menu entry opening a terminal")
                    iconName: "terminal"
                    onClicked: {
                        overflowMenu.close();
                        topbar.terminalRequested();
                    }
                }
            }
        }
    }

    DankSearchField {
        id: search

        anchors.left: leftActions.right
        anchors.right: rightActions.left
        anchors.leftMargin: FilesMetrics.topbarSpacing
        anchors.rightMargin: FilesMetrics.topbarSpacing
        anchors.verticalCenter: parent.verticalCenter
        controlHeight: FilesMetrics.topbarControlSize
        backgroundColor: Theme.surfaceContainerLowest
        placeholderText: I18n.tr("Search in Files", "placeholder in the search field")
    }
}
