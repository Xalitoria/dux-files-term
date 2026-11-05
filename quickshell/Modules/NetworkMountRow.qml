import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.FileBrowser

FilePlaceRow {
    id: row

    required property var modelData
    required property var host

    readonly property string target: modelData.homePath || modelData.path

    label: modelData.name
    iconName: modelData.iconName
    tooltipText: [modelData.name, modelData.uri, target].filter((line, index, lines) => line && lines.indexOf(line) === index).join("\n")
    actionIcon: "eject"
    actionTooltip: I18n.tr("Disconnect", "network mount action")
    busy: host.busyId === modelData.id
    dropPath: target
    onClicked: host.navigateRequested(target)
    onMiddleClicked: host.tabRequested(target)
    onDropped: drop => host.dropRequested(target, drop)
    onSpringLoaded: host.navigateRequested(target)
    onActionTriggered: {
        host.busyId = modelData.id;
        NetworkService.disconnect(modelData.id, () => host.busyId = "");
    }
}
