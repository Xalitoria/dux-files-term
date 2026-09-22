pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Models
import qs.DankCommon.Widgets

DankListView {
    id: list

    // The contract both file views implement: FileViewBody drives it, and the
    // delegates read it back through their view property.
    property SelectionModel selection: null
    property real iconSize: Theme.iconSize
    property string renamingPath: ""
    property var cutSet: ({})
    property var childCounts: ({})
    property var columnIds: []
    property var storedWidths: ({})

    readonly property int columnsPerRow: 1
    readonly property real rowHeight: Math.max(Theme.listItemHeight - Theme.spacingM, iconSize + Theme.spacingM * 2)
    readonly property real rowPitch: rowHeight + spacing
    readonly property real contentOrigin: headerItem?.height ?? 0

    signal itemClicked(int index, int modifiers)
    signal itemActivated(int index)
    signal itemContextMenu(int index, real pointX, real pointY, int modifiers)
    signal itemDragStarted(int index)
    signal itemDropped(int index, var drop)
    signal renameSubmitted(string path, string name)
    signal renameCancelled

    function columnWidth(id) {
        return storedWidths[id] ?? FileColumns.specFor(id)?.width ?? FilesMetrics.columnMinWidth;
    }

    function visibleRange() {
        const first = Math.floor((contentY - contentOrigin) / rowPitch);
        const last = Math.floor((contentY - contentOrigin + height) / rowPitch);
        return [Math.max(0, first), Math.min(count - 1, last)];
    }

    // Geometry is computed, not read off instantiated delegates, so a rubber
    // band still covers rows the view has not created yet.
    function indicesIn(rect) {
        const first = Math.max(0, Math.floor((rect.y - contentOrigin) / rowPitch));
        const last = Math.min(count - 1, Math.floor((rect.y + rect.height - contentOrigin) / rowPitch));
        const out = [];
        for (let i = first; i <= last; i++)
            out.push(i);
        return out;
    }

    clip: true
    spacing: Theme.groupedListGap
    cacheBuffer: Math.max(0, height)
    currentIndex: -1
    reuseItems: true

    delegate: FileRow {
        view: list
    }
}
