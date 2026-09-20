pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

OverlayDialog {
    id: root

    property string path: ""
    property string name: ""

    signal runRequested
    signal editRequested

    function ask(entryPath, entryName) {
        path = entryPath;
        name = entryName;
        open();
    }

    title: I18n.tr("Run this file?", "executable file dialog header")
    supportingText: I18n.tr("%1 can be run as a program. Open it in an editor instead to read it first.", "executable file dialog explanation").arg(name)
    onAccepted: {
        runRequested();
        close();
    }

    actions: [
        DankButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: root.close()
        },
        DankButton {
            text: I18n.tr("Open in editor", "executable file dialog action opening the file as text")
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: {
                root.editRequested();
                root.close();
            }
        },
        DankButton {
            text: I18n.tr("Run", "executable file dialog action running the file")
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: {
                root.runRequested();
                root.close();
            }
        }
    ]
}
