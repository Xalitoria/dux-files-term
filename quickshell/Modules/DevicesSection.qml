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
    signal unlockRequested(var device)
    signal dropRequested(string path, var drop)

    function activate(device) {
        if (device.encrypted && !device.unlocked) {
            unlockRequested(device);
            return;
        }
        if (device.mounted) {
            navigateRequested(device.mountPoint);
            return;
        }

        busyId = device.id;
        DevicesService.mount(device.id, result => {
            section.busyId = "";
            if (result.error)
                return;
            section.navigateRequested(result.mountPoint);
        });
    }

    function detach(device) {
        busyId = device.id;
        const done = () => section.busyId = "";
        if (device.encrypted && device.unlocked) {
            DevicesService.lock(device.id, done);
            return;
        }
        if (device.ejectable || device.canPowerOff) {
            DevicesService.eject(device.id, done);
            return;
        }
        DevicesService.unmount(device.id, false, done);
    }

    function actionIconFor(device) {
        switch (true) {
        case device.system:
            return "";
        case device.encrypted && !device.unlocked:
            return "lock_open";
        case !device.mounted:
            return "play_arrow";
        case device.ejectable || device.canPowerOff:
            return "eject";
        default:
            return "stop_circle";
        }
    }

    title: I18n.tr("Devices", "sidebar section listing drives and volumes")
    emptyText: {
        if (DevicesService.available)
            return "";
        if (!SandboxService.loaded)
            return "";
        if (!SandboxService.available("devices"))
            return FilesFormat.featureUnavailable(SandboxService.reason("devices"));
        return I18n.tr("Drives are unavailable because udisks2 is not running", "sidebar explanation when the device service is missing");
    }
    showEmpty: !DevicesService.available

    function activateOrMount(device) {
        if (device.mounted || device.encrypted || SettingsData.mountOnClick) {
            activate(device);
            return;
        }
        busyId = device.id;
        DevicesService.mount(device.id, () => section.busyId = "");
    }

    Repeater {
        model: DevicesService.devices.filter(device => !device.hidden || SettingsData.showSystemPartitions)

        FilePlaceRow {
            required property var modelData

            label: modelData.system ? I18n.tr("My Computer", "sidebar entry for the drive the system runs from") : modelData.name
            iconName: modelData.encrypted && !modelData.unlocked ? "lock" : modelData.iconName
            ringValue: modelData.mounted ? DevicesService.usageRatio(modelData.mountPoint) : -1
            actionIcon: section.actionIconFor(modelData)
            busy: section.busyId === modelData.id
            dropPath: modelData.mounted ? modelData.mountPoint : ""
            onClicked: section.activateOrMount(modelData)
            onDropped: drop => section.dropRequested(modelData.mountPoint, drop)
            onActionTriggered: {
                if (!modelData.mounted || modelData.system) {
                    section.activate(modelData);
                    return;
                }
                section.detach(modelData);
            }
        }
    }

    Repeater {
        model: NetworkService.portables

        FilePlaceRow {
            required property var modelData

            label: modelData.name
            iconName: modelData.iconName
            actionIcon: "play_arrow"
            busy: section.busyId === modelData.id
            onClicked: {
                section.busyId = modelData.id;
                NetworkService.mountPortable(modelData.monitor, modelData.id, () => section.busyId = "");
            }
            onActionTriggered: clicked()
        }
    }

    Repeater {
        model: NetworkService.deviceMounts

        FilePlaceRow {
            required property var modelData

            label: modelData.name
            iconName: modelData.iconName
            actionIcon: "eject"
            busy: section.busyId === modelData.id
            dropPath: modelData.path
            onClicked: section.navigateRequested(modelData.path)
            onDropped: drop => section.dropRequested(modelData.path, drop)
            onActionTriggered: {
                section.busyId = modelData.id;
                NetworkService.disconnect(modelData.id, () => section.busyId = "");
            }
        }
    }
}
