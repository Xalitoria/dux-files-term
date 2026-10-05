pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

FilePlaceSection {
    id: section

    property string currentPath: ""

    signal navigateRequested(string path)
    signal dropRequested(string path, var drop)
    signal menuRequested(string path, real pointX, real pointY)

    readonly property var places: DirectoryService.userDirs.map(dir => ({
                "path": dir.path,
                "label": dir.key === "home" ? I18n.tr("Home", "sidebar entry for the user's home folder") : dir.name,
                "iconName": dir.iconName
            }))
    readonly property var placePaths: places.map(place => place.path)
    readonly property var bookmarks: BookmarksService.entries.filter(entry => !placePaths.includes(entry.path))

    title: I18n.tr("Bookmarks", "sidebar section listing saved folders")

    Repeater {
        model: section.places

        FilePlaceRow {
            required property var modelData

            label: modelData.label
            iconName: modelData.iconName
            selected: modelData.path === section.currentPath
            dropPath: modelData.path
            onClicked: section.navigateRequested(modelData.path)
            onDropped: drop => section.dropRequested(modelData.path, drop)
        }
    }

    DReorderList {
        id: reorderList

        width: section.width
        model: section.bookmarks
        spacing: Theme.spacingXXS
        onReordered: indices => BookmarksService.reorder(indices.map(index => section.bookmarks[index].path).concat(BookmarksService.paths.filter(path => section.placePaths.includes(path))))

        delegate: FilePlaceRow {
            id: row

            required property var modelData
            required property int index

            readonly property bool dragging: reorderList.draggingIndex === index

            width: reorderList.width
            label: modelData.label
            iconName: modelData.icon || "folder"
            selected: modelData.path === section.currentPath
            actionIcon: "bookmark_remove"
            actionTooltip: I18n.tr("Remove bookmark", "tooltip on the bookmark button")
            dropPath: modelData.path
            z: dragging ? 1 : 0
            opacity: dragging ? Theme.pendingOpacity : 1
            onClicked: section.navigateRequested(modelData.path)
            onActionTriggered: BookmarksService.remove(modelData.path)
            onDropped: drop => section.dropRequested(modelData.path, drop)

            Behavior on y {
                enabled: reorderList.animateLayout && !row.dragging && Theme.animationsEnabled
                NumberAnimation {
                    duration: Theme.expressiveDurations.expressiveDefaultSpatial
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Theme.expressiveCurves.expressiveFastSpatial
                }
            }

            TapHandler {
                acceptedButtons: Qt.RightButton
                onTapped: point => {
                    const mapped = row.mapToItem(section, point.position.x, point.position.y);
                    section.menuRequested(row.modelData.path, mapped.x, mapped.y);
                }
            }

            DragHandler {
                target: null
                xAxis.enabled: false
                enabled: reorderList.count > 1
                onGrabChanged: (transition, point) => {
                    switch (transition) {
                    case PointerDevice.GrabExclusive:
                        reorderList.begin(row.index, reorderList.mapFromItem(null, point.scenePressPosition.x, point.scenePressPosition.y));
                        reorderList.dragTo(reorderList.mapFromItem(null, point.scenePosition.x, point.scenePosition.y));
                        return;
                    case PointerDevice.UngrabExclusive:
                        reorderList.finish();
                        return;
                    case PointerDevice.CancelGrabExclusive:
                        reorderList.cancel();
                        return;
                    }
                }
                onCentroidChanged: {
                    if (!active || !row.dragging)
                        return;
                    reorderList.dragTo(reorderList.mapFromItem(null, centroid.scenePosition.x, centroid.scenePosition.y));
                }
            }
        }
    }
}
