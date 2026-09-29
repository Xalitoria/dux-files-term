import QtQuick
import qs.Common
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

OverlayDialog {
    id: root

    property var paths: []
    property string format: "zip"

    readonly property var formats: [
        {
            "value": "zip",
            "label": ".zip"
        },
        {
            "value": "tar.gz",
            "label": ".tar.gz"
        }
    ]

    signal submitted(var paths, string name, string format)

    function ask(targets) {
        paths = targets;
        const first = targets[0] ?? "";
        const base = first.substring(first.lastIndexOf("/") + 1);
        nameField.text = targets.length === 1 ? base.replace(/(\.tar)?\.[^.]+$/, "") || base : I18n.tr("Archive", "default name for a new archive");
        open();
        nameField.forceActiveFocus();
        nameField.selectAll();
    }

    function submit() {
        const trimmed = nameField.text.trim();
        if (trimmed === "" || paths.length === 0)
            return;
        submitted(paths, trimmed, format);
        close();
    }

    title: I18n.tr("Compress", "compress dialog header")
    supportingText: paths.length === 1 ? paths[0].substring(paths[0].lastIndexOf("/") + 1) : FileFormat.count(paths.length)
    onAccepted: submit()

    DankTextField {
        id: nameField

        width: parent.width
        outlined: true
        labelText: I18n.tr("Name", "label for a file name input")
        onAccepted: root.submit()
        Keys.onReturnPressed: event => event.accepted = true
        Keys.onEnterPressed: event => event.accepted = true
    }

    DankFilterChips {
        width: parent.width
        model: root.formats
        currentIndex: Math.max(0, root.formats.findIndex(entry => entry.value === root.format))
        showCounts: false
        onSelectionChanged: index => root.format = root.formats[index].value
    }

    actions: [
        TextButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            onClicked: root.close()
        },
        DankButton {
            text: I18n.tr("Create", "dialog button creating an item")
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: root.submit()
        }
    ]
}
