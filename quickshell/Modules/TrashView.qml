pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser
import "../DCommon/FileBrowser/NameFilter.js" as NameFilter

Rectangle {
    id: trashView

    property string viewMode: "list"
    property int gridZoom: 1
    property int listZoom: 0
    property var columnWidths: ({})
    property bool previewOpen: false
    property real previewWidth: 0
    property string nameFilter: ""
    property bool singleClickActivates: false
    property bool typeAheadEnabled: true
    property string typeAheadMode: "jump"
    property bool countFolders: true
    property bool thumbnailsEnabled: true
    property real thumbnailSizeLimit: -1
    property bool outlined: false

    readonly property alias selection: selectionModel
    readonly property alias body: viewBody
    readonly property alias model: trashModel
    readonly property bool compact: width < Theme.smallBreakpoint
    readonly property string restoreLabel: I18n.tr("Restore", "trash action returning items to where they were deleted from")
    readonly property string emptyLabel: I18n.tr("Empty Trash", "action deleting everything in the trash")
    readonly property string summary: {
        if (trashModel.error !== "")
            return FileFormat.listingError(trashModel.errorCode);
        if (trashModel.total === 0)
            return "";
        if (nameFilter !== "")
            return I18n.tr("%1 matching “%2”", "filter bar above a folder listing, count then the typed text").arg(FileFormat.count(trashModel.count)).arg(nameFilter);
        return FileFormat.count(trashModel.total);
    }

    signal activateRequested(var entry)
    signal itemMenuRequested(int index, real pointX, real pointY, int modifiers)
    signal backgroundMenuRequested(real pointX, real pointY)
    signal restoreRequested
    signal emptyRequested
    signal itemsDropped(var drop)
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
        TrashService.refresh();
    }

    radius: FilesMetrics.paneRadius
    color: Theme.foregroundColor(Theme.cardSurface, true)
    border.width: outlined ? Theme.outlineWidthFocused : 0
    border.color: Theme.primary

    TrashModel {
        id: trashModel

        nameFilter: trashView.nameFilter
        thumbnailsEnabled: trashView.thumbnailsEnabled
        thumbnailSizeLimit: trashView.thumbnailSizeLimit
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
                text: I18n.tr("Trash", "trash view header and sidebar entry")
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Theme.fontWeightMedium
                elide: Text.ElideRight
            }

            StyledText {
                width: parent.width
                text: trashView.summary
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
                compact: trashView.compact
                label: trashView.restoreLabel
                iconName: "restore_from_trash"
                enabled: selectionModel.count > 0
                onTriggered: {
                    trashView.focusRequested();
                    trashView.restoreRequested();
                }
            }

            HeaderAction {
                compact: trashView.compact
                label: trashView.emptyLabel
                iconName: "delete_forever"
                containerColor: Theme.errorContainer
                contentColor: Theme.onErrorContainer
                enabled: TrashService.count > 0
                onTriggered: {
                    trashView.focusRequested();
                    trashView.emptyRequested();
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
        directory: trashModel
        selection: selectionModel
        viewMode: trashView.viewMode === "columns" ? "list" : trashView.viewMode
        gridZoom: trashView.gridZoom
        listZoom: trashView.listZoom
        listColumns: ["origin", "deleted", "size"]
        columnWidths: trashView.columnWidths
        previewKeys: true
        sortKey: trashModel.sortKey
        sortDescending: trashModel.sortDescending
        showHidden: true
        dragEnabled: false
        dropEnabled: false
        showSelectionFooter: true
        emptyText: trashView.nameFilter !== "" ? I18n.tr("No results", "search results header when nothing matched") : I18n.tr("Trash is empty", "trash view placeholder and sidebar tooltip when nothing is trashed")
        surfaceColor: trashView.color
        singleClickActivates: trashView.singleClickActivates
        typeAheadEnabled: trashView.typeAheadEnabled
        typeAheadMode: trashView.typeAheadMode
        countFolders: trashView.countFolders
        nameHighlight: NameFilter.needle(trashView.nameFilter)
        filtering: trashView.nameFilter !== ""
        onFilterRequested: text => trashView.filterRequested(text)
        onFilterFocusRequested: trashView.filterFocusRequested()
        onActivateRequested: entry => trashView.activateRequested(entry)
        onItemMenuRequested: (index, pointX, pointY, modifiers) => {
            trashView.focusRequested();
            const point = mapToItem(trashView, pointX, pointY);
            trashView.itemMenuRequested(index, point.x, point.y, modifiers);
        }
        onBackgroundMenuRequested: (pointX, pointY) => {
            trashView.focusRequested();
            const point = mapToItem(trashView, pointX, pointY);
            trashView.backgroundMenuRequested(point.x, point.y);
        }
        onSortRequested: key => trashModel.sortBy(key)
        onZoomRequested: (mode, level) => trashView.zoomRequested(mode, level)
        onColumnResizeRequested: (id, width) => trashView.columnResizeRequested(id, width)
        onPreviewKeyRequested: (action, step) => {
            if (action === "toggle") {
                trashView.previewToggleRequested();
                return;
            }
            drawer.handleKey(action, step);
        }
        onFocusRequested: trashView.focusRequested()
    }

    FilePreviewDrawer {
        id: drawer

        anchors.top: viewBody.top
        anchors.bottom: viewBody.bottom
        anchors.right: parent.right
        anchors.rightMargin: FilesMetrics.paneBodyMarginH
        body: viewBody
        open: trashView.previewOpen
        available: trashView.width - FilesMetrics.paneBodyMarginH * 2
        preferredWidth: trashView.previewWidth
        onResized: width => trashView.previewResized(width)
        onFocusRequested: trashView.focusRequested()
    }

    Rectangle {
        anchors.fill: parent
        radius: parent.radius
        visible: dropZone.containsDrag
        color: Theme.withAlpha(Theme.primary, Theme.stateLayerDrag)
    }

    FileDropArea {
        id: dropZone

        anchors.fill: viewBody
        dropPath: TrashService.filesPath
        enabled: dropPath !== ""
        onDropped: drop => trashView.itemsDropped(drop)
    }

    TapHandler {
        acceptedButtons: Qt.BackButton | Qt.ForwardButton
        onTapped: (point, button) => {
            trashView.focusRequested();
            if (button === Qt.BackButton) {
                trashView.backRequested();
                return;
            }
            trashView.forwardRequested();
        }
    }
}
