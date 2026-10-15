pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

// The pane body while a recursive search is showing: a header naming the
// query and its root, the filter chips, then the shared list over SearchModel.
Rectangle {
    id: results

    required property SearchModel model
    property string viewMode: "list"
    property int gridZoom: 1
    property int listZoom: 0
    property var columnWidths: ({})
    property bool previewOpen: false
    property real previewWidth: 0
    property var cutSet: ({})
    property bool singleClickActivates: false
    property bool typeAheadEnabled: true
    property bool countFolders: true

    readonly property alias selection: selectionModel
    readonly property alias body: viewBody
    readonly property alias chips: chips
    readonly property string summary: {
        if (model.error !== "")
            return FileFormat.listingError(model.errorCode);
        if (model.loading)
            return I18n.tr("Searching…", "search results header while the daemon walks the tree");
        if (model.truncated)
            return I18n.tr("Showing the first %1 results", "search results header when the cap was hit").arg(model.count) + (model.unreadable > 0 ? " · " + unreadableText : "");
        if (model.count === 0)
            return I18n.tr("No results", "search results header when nothing matched");
        return FileFormat.count(model.count) + (model.unreadable > 0 ? " · " + unreadableText : "");
    }
    readonly property string unreadableText: I18n.tr("%1 folders could not be read", "search results header note").arg(model.unreadable)

    signal activateRequested(var entry)
    signal itemMenuRequested(int index, real pointX, real pointY, int modifiers)
    signal dragRequested(var paths)
    signal renameRequested(string target, string name)
    signal zoomRequested(string viewMode, int level)
    signal columnResizeRequested(string id, real width)
    signal previewToggleRequested
    signal previewResized(real width)
    signal focusRequested
    signal tabRequested(string path)
    signal closeRequested
    signal filtersChanged

    function focusBody() {
        viewBody.focusBody();
    }

    radius: FilesMetrics.paneRadius
    color: Theme.foregroundColor(Theme.cardSurface, true)

    SelectionModel {
        id: selectionModel
    }

    Item {
        id: header

        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.topMargin: FilesMetrics.paneBodyMargin
        anchors.leftMargin: FilesMetrics.paneBodyMarginH
        anchors.rightMargin: FilesMetrics.paneBodyMarginH
        height: Math.max(FileBrowserMetrics.pathPillHeight, titleColumn.implicitHeight)

        Column {
            id: titleColumn

            anchors.left: parent.left
            anchors.right: closeButton.left
            anchors.rightMargin: Theme.spacingS
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingXXS

            StyledText {
                width: parent.width
                text: results.model.query !== "" ? I18n.tr("Results for “%1” in %2", "search results header, query then folder").arg(results.model.query).arg(Paths.shortenHome(results.model.rootPath)) : I18n.tr("Results in %1", "search results header without a query").arg(Paths.shortenHome(results.model.rootPath))
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Theme.fontWeightMedium
                elide: Text.ElideMiddle
            }

            StyledText {
                width: parent.width
                text: results.summary
                color: Theme.surfaceVariantText
                font.pixelSize: Theme.fontSizeSmall
                elide: Text.ElideRight
            }
        }

        DActionButton {
            id: closeButton

            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            buttonSize: Theme.buttonHeightXS
            iconName: "close"
            iconSize: Theme.iconSizeSmall
            iconColor: Theme.surfaceVariantText
            tooltipText: I18n.tr("Close", "dialog button to close")
            onClicked: results.closeRequested()
        }
    }

    SearchChips {
        id: chips

        anchors.top: header.bottom
        anchors.topMargin: Theme.spacingS
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.leftMargin: FilesMetrics.paneBodyMarginH
        anchors.rightMargin: FilesMetrics.paneBodyMarginH
        onChanged: results.filtersChanged()
    }

    FileViewBody {
        id: viewBody

        anchors.top: chips.bottom
        anchors.topMargin: Theme.spacingS
        anchors.bottom: parent.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottomMargin: FilesMetrics.paneBodyMargin
        anchors.leftMargin: FilesMetrics.paneBodyMarginH
        anchors.rightMargin: FilesMetrics.paneBodyMarginH + drawer.width
        focus: true
        directory: results.model
        selection: selectionModel
        viewMode: results.viewMode === "columns" ? "list" : results.viewMode
        gridZoom: results.gridZoom
        listZoom: results.listZoom
        listColumns: ["location", "size", "modified"]
        columnWidths: results.columnWidths
        previewKeys: true
        sortKey: ""
        showHidden: chips.includeHidden
        dragEnabled: true
        dropEnabled: false
        showSelectionFooter: true
        emptyText: ""
        surfaceColor: results.color
        cutSet: results.cutSet
        singleClickActivates: results.singleClickActivates
        typeAheadEnabled: results.typeAheadEnabled
        countFolders: results.countFolders
        nameHighlight: results.model.query
        onActivateRequested: entry => results.activateRequested(entry)
        onItemMenuRequested: (index, pointX, pointY, modifiers) => {
            results.focusRequested();
            const point = mapToItem(results, pointX, pointY);
            results.itemMenuRequested(index, point.x, point.y, modifiers);
        }
        onDragRequested: paths => results.dragRequested(paths)
        onRenameRequested: (target, name) => results.renameRequested(target, name)
        onZoomRequested: (mode, level) => results.zoomRequested(mode, level)
        onColumnResizeRequested: (id, width) => results.columnResizeRequested(id, width)
        onPreviewKeyRequested: (action, step) => {
            if (action === "toggle") {
                results.previewToggleRequested();
                return;
            }
            drawer.handleKey(action, step);
        }
        onFocusRequested: results.focusRequested()
        onTabRequested: target => results.tabRequested(target)
        onOpenRequested: target => {
            if (target === "..")
                return;
            results.focusRequested();
            results.activateRequested({
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
        open: results.previewOpen
        available: results.width - FilesMetrics.paneBodyMarginH * 2
        preferredWidth: results.previewWidth
        onResized: width => results.previewResized(width)
        onFocusRequested: results.focusRequested()
    }
}
