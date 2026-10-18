pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser
import "../DCommon/FileBrowser/NameFilter.js" as NameFilter

Rectangle {
    id: recentView

    property string viewMode: "list"
    property int gridZoom: 1
    property int listZoom: 0
    property var columnWidths: ({})
    property bool previewOpen: false
    property real previewWidth: 0
    property var cutSet: ({})
    property string nameFilter: ""
    property bool singleClickActivates: false
    property bool typeAheadEnabled: true
    property string typeAheadMode: "jump"
    property bool countFolders: true
    property bool thumbnailsEnabled: true
    property real thumbnailSizeLimit: -1
    property bool dragEnabled: true
    property bool outlined: false

    readonly property alias selection: selectionModel
    readonly property alias body: viewBody
    readonly property alias model: recentModel
    readonly property bool compact: width < Theme.smallBreakpoint
    readonly property string removeLabel: I18n.tr("Remove from Recent", "action dropping the selected files from the recently used list without touching the files")
    readonly property string clearLabel: I18n.tr("Clear Recent", "action emptying the recently used list without touching the files")
    readonly property string summary: {
        if (recentModel.error !== "")
            return FileFormat.listingError(recentModel.errorCode);
        if (recentModel.total === 0)
            return "";
        if (nameFilter !== "")
            return I18n.tr("%1 matching “%2”", "filter bar above a folder listing, count then the typed text").arg(FileFormat.count(recentModel.count)).arg(nameFilter);
        return FileFormat.count(recentModel.total);
    }

    signal activateRequested(var entry)
    signal itemMenuRequested(int index, real pointX, real pointY, int modifiers)
    signal backgroundMenuRequested(real pointX, real pointY)
    signal removeRequested
    signal clearRequested
    signal dragRequested(var paths)
    signal tabRequested(string path)
    signal zoomRequested(string viewMode, int level)
    signal columnResizeRequested(string id, real width)
    signal previewToggleRequested
    signal previewResized(real width)
    signal focusRequested
    signal backRequested
    signal forwardRequested
    signal filterRequested(string text)
    signal filterFocusRequested

    function focusBody() {
        viewBody.focusBody();
    }

    function refresh() {
        recentModel.refresh();
    }

    radius: FilesMetrics.paneRadius
    color: Theme.foregroundColor(Theme.cardSurface, true)
    border.width: outlined ? Theme.outlineWidthFocused : 0
    border.color: Theme.primary

    component HeaderAction: Item {
        id: headerAction

        property string label: ""
        property string iconName: ""

        signal triggered

        implicitWidth: recentView.compact ? iconButton.width : textButton.implicitWidth
        implicitHeight: Theme.buttonHeightXS

        DButton {
            id: textButton

            visible: !recentView.compact
            text: headerAction.label
            iconName: headerAction.iconName
            buttonHeight: Theme.buttonHeightXS
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: headerAction.triggered()
        }

        DActionButton {
            id: iconButton

            visible: recentView.compact
            buttonSize: Theme.buttonHeightXS
            iconName: headerAction.iconName
            iconSize: Theme.iconSizeMedium
            iconColor: Theme.onSecondaryContainer
            backgroundColor: Theme.secondaryContainer
            tooltipText: headerAction.label
            onClicked: headerAction.triggered()
        }
    }

    RecentModel {
        id: recentModel

        nameFilter: recentView.nameFilter
        thumbnailsEnabled: recentView.thumbnailsEnabled
        thumbnailSizeLimit: recentView.thumbnailSizeLimit
    }

    SelectionModel {
        id: selectionModel
    }

    Item {
        id: heading

        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.topMargin: FilesMetrics.paneBodyMargin
        anchors.leftMargin: FilesMetrics.paneBodyMarginH
        anchors.rightMargin: FilesMetrics.paneBodyMarginH
        height: Math.max(FileBrowserMetrics.pathPillHeight, titleColumn.implicitHeight, actions.implicitHeight)

        Column {
            id: titleColumn

            anchors.left: parent.left
            anchors.right: actions.left
            anchors.rightMargin: Theme.spacingS
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingXXS

            StyledText {
                width: parent.width
                text: I18n.tr("Recent", "recent files view header and sidebar entry")
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Theme.fontWeightMedium
                elide: Text.ElideRight
            }

            StyledText {
                width: parent.width
                text: recentView.summary
                color: Theme.surfaceVariantText
                font.pixelSize: Theme.fontSizeSmall
                elide: Text.ElideRight
            }
        }

        Row {
            id: actions

            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS

            HeaderAction {
                label: recentView.removeLabel
                iconName: "delete_history"
                enabled: selectionModel.count > 0
                onTriggered: {
                    recentView.focusRequested();
                    recentView.removeRequested();
                }
            }

            HeaderAction {
                label: recentView.clearLabel
                iconName: "delete_sweep"
                enabled: recentModel.total > 0
                onTriggered: {
                    recentView.focusRequested();
                    recentView.clearRequested();
                }
            }
        }
    }

    FileViewBody {
        id: viewBody

        anchors.top: heading.bottom
        anchors.topMargin: Theme.spacingS
        anchors.bottom: parent.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottomMargin: FilesMetrics.paneBodyMargin
        anchors.leftMargin: FilesMetrics.paneBodyMarginH
        anchors.rightMargin: FilesMetrics.paneBodyMarginH + drawer.width
        focus: true
        directory: recentModel
        selection: selectionModel
        viewMode: recentView.viewMode === "columns" ? "list" : recentView.viewMode
        gridZoom: recentView.gridZoom
        listZoom: recentView.listZoom
        listColumns: ["location", "used", "size"]
        columnWidths: recentView.columnWidths
        columnSortKeys: ({
                "location": "location"
            })
        previewKeys: true
        sortKey: recentModel.sortKey
        sortDescending: recentModel.sortDescending
        showHidden: true
        dragEnabled: recentView.dragEnabled
        dropEnabled: false
        showSelectionFooter: true
        emptyText: recentView.nameFilter !== "" ? I18n.tr("No results", "search results header when nothing matched") : I18n.tr("No recent files", "recent files view placeholder when nothing was opened yet")
        surfaceColor: recentView.color
        cutSet: recentView.cutSet
        singleClickActivates: recentView.singleClickActivates
        typeAheadEnabled: recentView.typeAheadEnabled
        typeAheadMode: recentView.typeAheadMode
        countFolders: recentView.countFolders
        nameHighlight: NameFilter.needle(recentView.nameFilter)
        filtering: recentView.nameFilter !== ""
        onFilterRequested: text => recentView.filterRequested(text)
        onFilterFocusRequested: recentView.filterFocusRequested()
        onActivateRequested: entry => recentView.activateRequested(entry)
        onItemMenuRequested: (index, pointX, pointY, modifiers) => {
            recentView.focusRequested();
            const point = mapToItem(recentView, pointX, pointY);
            recentView.itemMenuRequested(index, point.x, point.y, modifiers);
        }
        onBackgroundMenuRequested: (pointX, pointY) => {
            recentView.focusRequested();
            const point = mapToItem(recentView, pointX, pointY);
            recentView.backgroundMenuRequested(point.x, point.y);
        }
        onDragRequested: paths => recentView.dragRequested(paths)
        onSortRequested: key => recentModel.sortBy(key)
        onZoomRequested: (mode, level) => recentView.zoomRequested(mode, level)
        onColumnResizeRequested: (id, width) => recentView.columnResizeRequested(id, width)
        onPreviewKeyRequested: (action, step) => {
            if (action === "toggle") {
                recentView.previewToggleRequested();
                return;
            }
            drawer.handleKey(action, step);
        }
        onFocusRequested: recentView.focusRequested()
        onTabRequested: target => recentView.tabRequested(target)
        onOpenRequested: target => {
            if (target === "..")
                return;
            recentView.focusRequested();
            recentView.activateRequested({
                "path": target,
                "isDir": true
            });
        }
    }

    FilePreviewDrawer {
        id: drawer

        anchors.top: viewBody.top
        anchors.bottom: viewBody.bottom
        anchors.right: parent.right
        anchors.rightMargin: FilesMetrics.paneBodyMarginH
        body: viewBody
        open: recentView.previewOpen
        available: recentView.width - FilesMetrics.paneBodyMarginH * 2
        preferredWidth: recentView.previewWidth
        onResized: width => recentView.previewResized(width)
        onFocusRequested: recentView.focusRequested()
    }

    TapHandler {
        acceptedButtons: Qt.BackButton | Qt.ForwardButton
        onTapped: (point, button) => {
            recentView.focusRequested();
            if (button === Qt.BackButton) {
                recentView.backRequested();
                return;
            }
            recentView.forwardRequested();
        }
    }
}
