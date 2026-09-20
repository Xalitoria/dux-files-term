pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

DankFlickable {
    id: sidebar

    signal navigateRequested(string path)
    signal unlockRequested(var device)
    signal connectRequested

    contentHeight: sections.height
    clip: true
    onVisibleChanged: {
        if (visible)
            DevicesService.refreshUsage(false);
    }

    Column {
        id: sections

        width: sidebar.width
        spacing: Theme.spacingL

        DevicesSection {
            width: sections.width
            onNavigateRequested: path => sidebar.navigateRequested(path)
            onUnlockRequested: device => sidebar.unlockRequested(device)
        }

        NetworkSection {
            width: sections.width
            onNavigateRequested: path => sidebar.navigateRequested(path)
            onConnectRequested: sidebar.connectRequested()
        }
    }
}
