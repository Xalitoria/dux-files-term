pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

FilePlaceSection {
    id: section

    property string busyId: ""

    signal navigateRequested(string path)
    signal tabRequested(string path)
    signal dropRequested(string path, var drop)
    signal connectRequested

    title: I18n.tr("Network", "sidebar section listing remote locations")
    emptyText: {
        if (NetworkService.available)
            return "";
        if (!SandboxService.loaded)
            return "";
        if (!SandboxService.available("network"))
            return FilesFormat.featureUnavailable(SandboxService.reason("network"));
        return I18n.tr("Network locations are unavailable because gvfs is not running", "sidebar explanation when the network service is missing");
    }
    showEmpty: true

    Repeater {
        model: NetworkService.networkMounts

        FilePlaceRow {
            required property var modelData

            readonly property string target: modelData.homePath || modelData.path

            label: modelData.name
            iconName: modelData.iconName
            tooltipText: [modelData.name, modelData.uri, target].filter((line, index, lines) => line && lines.indexOf(line) === index).join("\n")
            actionIcon: "eject"
            actionTooltip: I18n.tr("Disconnect", "network mount action")
            busy: section.busyId === modelData.id
            dropPath: target
            onClicked: section.navigateRequested(target)
            onMiddleClicked: section.tabRequested(target)
            onDropped: drop => section.dropRequested(target, drop)
            onSpringLoaded: section.navigateRequested(target)
            onActionTriggered: {
                section.busyId = modelData.id;
                NetworkService.disconnect(modelData.id, () => section.busyId = "");
            }
        }
    }

    FilePlaceRow {
        visible: NetworkService.available
        label: I18n.tr("Connect to Server…", "sidebar entry opening the connect to server dialog")
        iconName: "folder-remote"
        busy: NetworkService.connecting
        onClicked: section.connectRequested()
    }
}
