pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

SidebarSection {
    id: section

    property string busyId: ""

    signal navigateRequested(string path)
    signal connectRequested

    title: I18n.tr("Network", "sidebar section listing remote locations")
    emptyText: {
        if (NetworkService.available)
            return "";
        if (!SandboxService.loaded)
            return "";
        if (!SandboxService.available("network"))
            return FileFormat.featureUnavailable(SandboxService.reason("network"));
        return I18n.tr("Network locations are unavailable because gvfs is not running", "sidebar explanation when the network service is missing");
    }
    showEmpty: true

    Repeater {
        model: NetworkService.networkMounts

        SidebarRow {
            required property var modelData

            label: modelData.name
            iconName: modelData.iconName
            actionIcon: "eject"
            busy: section.busyId === modelData.id
            onClicked: section.navigateRequested(modelData.path)
            onActionTriggered: {
                section.busyId = modelData.id;
                NetworkService.disconnect(modelData.id, () => section.busyId = "");
            }
        }
    }

    SidebarRow {
        visible: NetworkService.available
        label: I18n.tr("Connect to Server…", "sidebar entry opening the connect to server dialog")
        iconName: "folder-remote"
        busy: NetworkService.connecting
        onClicked: section.connectRequested()
    }
}
