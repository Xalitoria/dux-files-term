pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser
import "../DCommon/FileBrowser/PathCompletion.js" as PathCompletion

Item {
    id: topbar

    property bool splitView: false
    property bool bookmarked: false
    property bool backEnabled: false
    property bool forwardEnabled: false
    property bool compact: false
    property string viewMode: "list"
    property bool previewOpen: false
    property string sortKey: "name"
    property bool sortDescending: false
    property bool trashActive: false
    property string filterText: ""
    property string currentPath: ""
    property bool searchFocused: false

    readonly property bool bookmarkPlace: BookmarksService.isPlace(currentPath)
    readonly property string bookmarkLabel: {
        if (bookmarkPlace)
            return I18n.tr("Already in the sidebar", "tooltip on the bookmark button over a folder the sidebar lists on its own");
        return bookmarked ? I18n.tr("Remove bookmark", "tooltip on the bookmark button") : I18n.tr("Bookmark this folder", "tooltip on the bookmark button");
    }
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
    readonly property string previewLabel: I18n.tr("Preview", "view menu entry and shortcut showing or hiding the preview drawer")
    readonly property var currentViewMode: viewModes.find(mode => mode.value === viewMode) ?? viewModes[0]
    property bool resultsActive: false
    readonly property var resultSortKeys: [
        {
            "key": "",
            "icon": "sort",
            "label": I18n.tr("Relevance", "sort criterion for search results, best match first")
        },
        {
            "key": "name",
            "icon": "sort_by_alpha",
            "label": FileColumns.label("name")
        },
        {
            "key": "mtime",
            "icon": "history",
            "label": FileColumns.label("modified")
        },
        {
            "key": "size",
            "icon": "straighten",
            "label": FileColumns.label("size")
        },
        {
            "key": "location",
            "icon": "folder_open",
            "label": FileColumns.label("location")
        }
    ]
    readonly property var sortKeys: sortButton.defaultKeys
    readonly property var trashSortKeys: [
        {
            "key": "deleted",
            "icon": "delete",
            "label": FileColumns.label("deleted")
        },
        {
            "key": "name",
            "icon": "sort_by_alpha",
            "label": FileColumns.label("name")
        },
        {
            "key": "origin",
            "icon": "folder_open",
            "label": FileColumns.label("origin")
        },
        {
            "key": "size",
            "icon": "straighten",
            "label": FileColumns.label("size")
        }
    ]
    readonly property var recentSuggestions: searchFocused && search.text === "" ? FolderHistory.top(FileBrowserMetrics.suggestionRows, currentPath).map(match => PathCompletion.row(match, "visited")) : []

    signal sidebarToggled
    signal backRequested
    signal forwardRequested
    signal splitToggled
    signal bookmarkToggled
    signal viewModeSelected(string mode)
    signal previewToggled
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
    signal resultsRequested

    function focusSearch() {
        search.forceActiveFocus();
        search.selectAll();
    }

    function openFilter(text) {
        search.text = filterText + text;
        focusFilter();
    }

    function focusFilter() {
        search.forceActiveFocus();
        search.cursorPosition = search.text.length;
    }

    function searchContains(x, y) {
        return search.contains(search.mapFromItem(topbar, x, y));
    }

    function clearSearch() {
        search.text = "";
        searchCleared();
    }

    onRecentSuggestionsChanged: {
        if (recentSuggestions.length > 0)
            FolderHistory.verify(recentSuggestions.map(row => row.path));
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
            enabled: !topbar.trashActive && !topbar.bookmarkPlace
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

                FileMenuDivider {}

                FileMenuItem {
                    text: topbar.previewLabel
                    iconName: "view_sidebar"
                    checked: topbar.previewOpen
                    onClicked: {
                        viewMenu.close();
                        topbar.previewToggled();
                    }
                }
            }
        }

        FlatButton {
            anchors.verticalCenter: parent.verticalCenter
            visible: !topbar.compact
            iconName: "view_sidebar"
            iconColor: topbar.previewOpen ? Theme.onPrimary : Theme.surfaceText
            backgroundColor: topbar.previewOpen ? Theme.primary : Theme.chipSurface
            tooltipText: topbar.previewLabel
            onClicked: topbar.previewToggled()
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
            id: sortButton

            anchors.verticalCenter: parent.verticalCenter
            keys: topbar.resultsActive ? topbar.resultSortKeys : topbar.trashActive ? topbar.trashSortKeys : defaultKeys
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
                    enabled: !topbar.trashActive && !topbar.bookmarkPlace
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
                    text: topbar.previewLabel
                    iconName: "view_sidebar"
                    checked: topbar.previewOpen
                    onClicked: {
                        overflowMenu.close();
                        topbar.previewToggled();
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
                    enabled: !topbar.trashActive
                    text: I18n.tr("Properties", "menu entry showing details about the current folder")
                    iconName: "info"
                    onClicked: {
                        overflowMenu.close();
                        topbar.propertiesRequested();
                    }
                }

                FileMenuItem {
                    enabled: !topbar.trashActive
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
                    enabled: !topbar.trashActive
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
            const picked = recentPopup.highlightedRow();
            if (picked) {
                topbar.navigateRequested(picked.path);
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
            if (recentPopup.rows.length > 0) {
                recentPopup.move(1);
                break;
            }
            if (topbar.filterText === "")
                return;
            topbar.resultsRequested();
            break;
        case Qt.Key_Up:
            if (recentPopup.rows.length === 0)
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
        rows: topbar.recentSuggestions
        visible: topbar.recentSuggestions.length > 0
        onChosen: row => topbar.navigateRequested(row.path)
    }
}
