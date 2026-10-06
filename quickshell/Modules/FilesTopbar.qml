pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

Item {
    id: topbar

    property bool splitView: false
    property bool bookmarked: false
    property bool backEnabled: false
    property bool forwardEnabled: false
    property bool compact: false
    property string viewMode: "list"
    property string sortKey: "name"
    property bool sortDescending: false
    property string filterText: ""
    property string currentPath: ""
    property var recentDirs: []
    property bool searchFocused: false

    readonly property string bookmarkLabel: bookmarked ? I18n.tr("Remove bookmark", "tooltip on the bookmark button") : I18n.tr("Bookmark this folder", "tooltip on the bookmark button")
    readonly property var viewModes: [
        {
            "value": "list",
            "icon": "view_list",
            "label": I18n.tr("List view", "view mode showing one row per item")
        },
        {
            "value": "grid",
            "icon": "grid_view",
            "label": I18n.tr("Grid view", "view mode showing items as tiles")
        },
        {
            "value": "columns",
            "icon": "view_column",
            "label": I18n.tr("Column view", "view mode showing folders side by side")
        }
    ]
    readonly property var currentViewMode: viewModes.find(mode => mode.value === viewMode) ?? viewModes[0]
    readonly property var recentSuggestions: searchFocused && search.text === "" ? recentDirs.filter(path => path !== currentPath).slice(0, FileBrowserMetrics.suggestionRows) : []

    signal sidebarToggled
    signal backRequested
    signal forwardRequested
    signal splitToggled
    signal bookmarkToggled
    signal viewModeSelected(string mode)
    signal sortKeySelected(string key)
    signal sortDirectionToggled
    signal propertiesRequested
    signal newWindowRequested
    signal copyPathRequested
    signal reloadRequested
    signal terminalRequested
    signal settingsRequested
    signal shortcutsRequested
    signal filterChanged(string text)
    signal searchRequested(string query)
    signal searchCleared
    signal navigateRequested(string path)

    function focusSearch() {
        search.forceActiveFocus();
        search.selectAll();
    }

    function clearSearch() {
        search.text = "";
        searchCleared();
    }

    onFilterTextChanged: {
        if (search.text !== filterText)
            search.text = filterText;
    }

    implicitHeight: FilesMetrics.topbarControlSize
    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    component FlatButton: DActionButton {
        focusPolicy: Qt.TabFocus
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
        spacing: FilesMetrics.topbarSpacing

        FlatButton {
            anchors.verticalCenter: parent.verticalCenter
            visible: !topbar.compact
            iconName: "bookmark"
            iconFilled: topbar.bookmarked
            iconColor: topbar.bookmarked ? Theme.primary : Theme.surfaceText
            backgroundColor: Theme.chipSurface
            tooltipText: topbar.bookmarkLabel
            onClicked: topbar.bookmarkToggled()
        }

        FlatButton {
            id: viewButton

            anchors.verticalCenter: parent.verticalCenter
            visible: !topbar.compact
            iconName: topbar.currentViewMode.icon
            backgroundColor: Theme.chipSurface
            tooltipText: I18n.tr("View", "keyboard shortcut group")
            onClicked: viewMenu.visible ? viewMenu.close() : viewMenu.open()

            FileMenu {
                id: viewMenu

                y: viewButton.height + Theme.spacingXS
                x: viewButton.width - width

                Repeater {
                    model: topbar.viewModes

                    FileMenuItem {
                        required property var modelData

                        text: modelData.label
                        iconName: modelData.icon
                        checked: topbar.viewMode === modelData.value
                        onClicked: {
                            viewMenu.close();
                            topbar.viewModeSelected(modelData.value);
                        }
                    }
                }
            }
        }

        FlatButton {
            anchors.verticalCenter: parent.verticalCenter
            visible: !topbar.compact
            iconName: "splitscreen_vertical_add"
            iconColor: topbar.splitView ? Theme.onPrimary : Theme.surfaceText
            backgroundColor: topbar.splitView ? Theme.primary : Theme.chipSurface
            tooltipText: I18n.tr("Split view", "tooltip on the button showing a second pane")
            onClicked: topbar.splitToggled()
        }

        FileSortButton {
            anchors.verticalCenter: parent.verticalCenter
            sortKey: topbar.sortKey
            descending: topbar.sortDescending
            showLabel: !topbar.compact
            onSortKeySelected: key => topbar.sortKeySelected(key)
            onDirectionToggled: topbar.sortDirectionToggled()
        }

        FlatButton {
            id: overflowButton

            anchors.verticalCenter: parent.verticalCenter
            iconName: "more_vert"
            backgroundColor: Theme.chipSurface
            tooltipText: I18n.tr("More actions", "tooltip on the overflow menu button")
            onClicked: overflowMenu.visible ? overflowMenu.close() : overflowMenu.open()

            FileMenu {
                id: overflowMenu

                y: overflowButton.height + Theme.spacingXS
                x: overflowButton.width - width

                FileMenuItem {
                    visible: topbar.compact
                    text: topbar.bookmarkLabel
                    iconName: topbar.bookmarked ? "bookmark_remove" : "bookmark"
                    onClicked: {
                        overflowMenu.close();
                        topbar.bookmarkToggled();
                    }
                }

                Repeater {
                    model: topbar.viewModes

                    FileMenuItem {
                        required property var modelData

                        visible: topbar.compact
                        text: modelData.label
                        iconName: modelData.icon
                        checked: topbar.viewMode === modelData.value
                        onClicked: {
                            overflowMenu.close();
                            topbar.viewModeSelected(modelData.value);
                        }
                    }
                }

                FileMenuItem {
                    visible: topbar.compact
                    text: I18n.tr("Split view", "tooltip on the button showing a second pane")
                    iconName: "splitscreen_vertical_add"
                    checked: topbar.splitView
                    onClicked: {
                        overflowMenu.close();
                        topbar.splitToggled();
                    }
                }

                FileMenuItem {
                    text: I18n.tr("New window", "menu entry opening another file manager window")
                    iconName: "open_in_new"
                    onClicked: {
                        overflowMenu.close();
                        topbar.newWindowRequested();
                    }
                }

                FileMenuItem {
                    text: I18n.tr("Properties", "menu entry showing details about the current folder")
                    iconName: "info"
                    onClicked: {
                        overflowMenu.close();
                        topbar.propertiesRequested();
                    }
                }

                FileMenuItem {
                    text: I18n.tr("Copy location path", "menu entry copying the current path to the clipboard")
                    iconName: "content_copy"
                    onClicked: {
                        overflowMenu.close();
                        topbar.copyPathRequested();
                    }
                }

                FileMenuItem {
                    text: I18n.tr("Reload", "menu entry re-reading the current folder")
                    iconName: "refresh"
                    onClicked: {
                        overflowMenu.close();
                        topbar.reloadRequested();
                    }
                }

                FileMenuItem {
                    visible: SandboxService.available("terminal")
                    text: I18n.tr("Open in terminal", "context menu entry opening a terminal")
                    iconName: "terminal"
                    onClicked: {
                        overflowMenu.close();
                        topbar.terminalRequested();
                    }
                }

                FileMenuItem {
                    text: I18n.tr("Keyboard shortcuts", "overlay listing every keyboard shortcut")
                    iconName: "keyboard"
                    onClicked: {
                        overflowMenu.close();
                        topbar.shortcutsRequested();
                    }
                }

                FileMenuItem {
                    text: I18n.tr("Settings", "settings window header title")
                    iconName: "settings"
                    onClicked: {
                        overflowMenu.close();
                        topbar.settingsRequested();
                    }
                }
            }
        }
    }

    DSearchField {
        id: search

        anchors.left: leftActions.right
        anchors.right: rightActions.left
        anchors.leftMargin: FilesMetrics.topbarSpacing
        anchors.rightMargin: FilesMetrics.topbarSpacing
        anchors.verticalCenter: parent.verticalCenter
        controlHeight: FilesMetrics.topbarControlSize
        leftIconSize: Theme.iconSizeMedium
        backgroundColor: Theme.surfaceContainerLowest
        placeholderText: I18n.tr("Search in Files", "placeholder in the search field")
        keyForwardTargets: [topbar]
        onFocusStateChanged: hasFocus => topbar.searchFocused = hasFocus
        onTextChanged: {
            recentPopup.highlightIndex = -1;
            if (text !== topbar.filterText)
                topbar.filterChanged(text);
        }
        onAccepted: {
            const picked = recentPopup.highlightedPath();
            if (picked !== "") {
                topbar.navigateRequested(picked);
                return;
            }
            if (text.trim() !== "")
                topbar.searchRequested(text.trim());
        }
        Keys.onReturnPressed: event => event.accepted = true
        Keys.onEnterPressed: event => event.accepted = true
    }

    Keys.onPressed: event => {
        switch (event.key) {
        case Qt.Key_Escape:
            topbar.clearSearch();
            break;
        case Qt.Key_Down:
            if (recentPopup.paths.length === 0)
                return;
            recentPopup.move(1);
            break;
        case Qt.Key_Up:
            if (recentPopup.paths.length === 0)
                return;
            recentPopup.move(-1);
            break;
        default:
            return;
        }
        event.accepted = true;
    }

    PathSuggestions {
        id: recentPopup

        anchorItem: search
        paths: topbar.recentSuggestions
        visible: topbar.recentSuggestions.length > 0
        onChosen: target => topbar.navigateRequested(target)
    }
}
