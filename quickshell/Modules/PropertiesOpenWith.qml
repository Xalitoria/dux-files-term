pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

Column {
    id: openWith

    property string mime: ""
    property string path: ""
    property var apps: []

    readonly property bool canSetDefault: SandboxService.available("setDefault")

    onMimeChanged: reload()
    Component.onCompleted: reload()

    function reload() {
        apps = [];
        if (mime === "")
            return;
        AppsService.appsForMime(mime, path, response => {
            if (response.error)
                return;
            openWith.apps = response.apps || [];
        });
    }

    spacing: Theme.spacingM

    StyledText {
        width: parent.width
        text: openWith.mime === "" ? I18n.tr("Not available", "explanation for a feature the environment does not permit") : openWith.mime
        color: Theme.surfaceVariantText
        font.pixelSize: Theme.fontSizeSmall
        elide: Text.ElideRight
    }

    StyledText {
        width: parent.width
        visible: !openWith.canSetDefault
        text: FileFormat.featureUnavailable("PORTALCHOOSER")
        color: Theme.surfaceVariantText
        font.pixelSize: Theme.fontSizeSmall
        wrapMode: Text.Wrap
    }

    DankListView {
        width: parent.width
        height: Theme.iconSize * 8
        clip: true
        spacing: Theme.groupedListGap
        model: openWith.apps

        delegate: AppRow {
            onClicked: {
                if (!openWith.canSetDefault)
                    return;
                AppsService.setDefault(openWith.mime, modelData.id, () => openWith.reload());
            }
        }
    }
}
