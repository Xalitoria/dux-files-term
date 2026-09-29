pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

DankFlickable {
    id: sidebar

    property string currentPath: ""

    signal navigateRequested(string path)
    signal unlockRequested(var device)
    signal connectRequested
    signal dropRequested(string path, var drop)
    signal bookmarkMenuRequested(string path, real pointX, real pointY)

    contentHeight: sections.height + FilesMetrics.sidebarPadding
    clip: true
    onVisibleChanged: {
        if (visible)
            DevicesService.refreshUsage(false);
    }

    Column {
        id: sections

        width: sidebar.width
        spacing: FilesMetrics.sidebarSectionSpacing

        BookmarksSection {
            id: bookmarksSection

            width: sections.width
            currentPath: sidebar.currentPath
            onNavigateRequested: path => sidebar.navigateRequested(path)
            onDropRequested: (path, drop) => sidebar.dropRequested(path, drop)
            onMenuRequested: (path, pointX, pointY) => {
                const point = bookmarksSection.mapToItem(sidebar, pointX, pointY);
                sidebar.bookmarkMenuRequested(path, point.x, point.y);
            }
        }

        DevicesSection {
            width: sections.width
            onNavigateRequested: path => sidebar.navigateRequested(path)
            onUnlockRequested: device => sidebar.unlockRequested(device)
            onDropRequested: (path, drop) => sidebar.dropRequested(path, drop)
        }

        NetworkSection {
            width: sections.width
            onNavigateRequested: path => sidebar.navigateRequested(path)
            onConnectRequested: sidebar.connectRequested()
            onDropRequested: (path, drop) => sidebar.dropRequested(path, drop)
        }
    }
}
