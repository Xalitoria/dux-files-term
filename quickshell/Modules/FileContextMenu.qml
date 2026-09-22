pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services

// One menu for both cases: with a selection it acts on it, without one it acts
// on the folder. Every row dispatches an action id, the same ids the keyboard
// map uses, so there is one place that decides what an action does.
ContextMenu {
    id: menu

    property int selectionCount: 0
    property bool hasDirectory: false
    property bool hasFile: false
    property bool hasClipboard: false
    property bool shiftHeld: false
    property bool forFolder: false

    signal actionTriggered(string id)

    readonly property var entries: forFolder ? backgroundRows : itemRows

    readonly property var itemRows: [
        {
            "id": "open",
            "icon": "open_in_new",
            "label": I18n.tr("Open", "context menu entry opening a file"),
            "enabled": selectionCount === 1
        },
        {
            "id": "openWith",
            "icon": "apps",
            "label": I18n.tr("Open with", "open with dialog header"),
            "enabled": hasFile
        },
        {
            "id": "newTab",
            "icon": "tab",
            "label": I18n.tr("Open in new tab", "context menu entry opening a folder in a new tab"),
            "enabled": hasDirectory && selectionCount === 1
        },
        {
            "id": "openInPane",
            "icon": "splitscreen_vertical_add",
            "label": I18n.tr("Open in the other pane", "context menu entry opening a folder beside the current one"),
            "enabled": hasDirectory && selectionCount === 1
        },
        {
            "id": "separator"
        },
        {
            "id": "cut",
            "icon": "content_cut",
            "label": I18n.tr("Cut", "context menu entry marking the selection for a move"),
            "enabled": selectionCount > 0
        },
        {
            "id": "copy",
            "icon": "content_copy",
            "label": I18n.tr("Copy", "context menu entry copying the selection"),
            "enabled": selectionCount > 0
        },
        {
            "id": "pasteInto",
            "icon": "content_paste",
            "label": I18n.tr("Paste into folder", "context menu entry pasting the clipboard into the selected folder"),
            "enabled": hasDirectory && selectionCount === 1 && hasClipboard
        },
        {
            "id": "rename",
            "icon": "edit",
            "label": I18n.tr("Rename", "rename dialog header"),
            "enabled": selectionCount === 1
        },
        {
            "id": "separator"
        },
        {
            "id": "trash",
            "icon": "delete",
            "label": I18n.tr("Move to Trash", "context menu entry trashing the selection"),
            "enabled": selectionCount > 0,
            "visible": !shiftHeld
        },
        {
            "id": "deletePermanently",
            "icon": "delete_forever",
            "label": I18n.tr("Delete permanently", "context menu entry deleting the selection for good"),
            "enabled": selectionCount > 0,
            "visible": shiftHeld,
            "danger": true
        },
        {
            "id": "copyPath",
            "icon": "link",
            "label": I18n.tr("Copy location path", "menu entry copying the current path to the clipboard"),
            "enabled": selectionCount > 0
        },
        {
            "id": "properties",
            "icon": "info",
            "label": I18n.tr("Properties", "menu entry showing details about the current folder"),
            "enabled": true
        }
    ]

    readonly property var backgroundRows: [
        {
            "id": "newFolder",
            "icon": "create_new_folder",
            "label": I18n.tr("New folder", "new folder dialog header"),
            "enabled": true
        },
        {
            "id": "paste",
            "icon": "content_paste",
            "label": I18n.tr("Paste", "context menu entry pasting the clipboard here"),
            "enabled": hasClipboard
        },
        {
            "id": "pasteLink",
            "icon": "add_link",
            "label": I18n.tr("Paste as link", "context menu entry creating symlinks from the clipboard"),
            "enabled": hasClipboard
        },
        {
            "id": "separator"
        },
        {
            "id": "selectAll",
            "icon": "select_all",
            "label": I18n.tr("Select all", "menu entry selecting every item in the folder"),
            "enabled": true
        },
        {
            "id": "showHidden",
            "icon": "visibility",
            "label": I18n.tr("Show hidden files", "menu entry listing dot files"),
            "enabled": true
        },
        {
            "id": "reload",
            "icon": "refresh",
            "label": I18n.tr("Reload", "menu entry re-reading the current folder"),
            "enabled": true
        },
        {
            "id": "openTerminal",
            "icon": "terminal",
            "label": I18n.tr("Open in terminal", "context menu entry opening a terminal"),
            "enabled": true,
            "visible": SandboxService.available("terminal")
        },
        {
            "id": "separator"
        },
        {
            "id": "properties",
            "icon": "info",
            "label": I18n.tr("Properties", "menu entry showing details about the current folder"),
            "enabled": true
        }
    ]

    Repeater {
        model: menu.entries

        Item {
            id: slot

            required property var modelData

            readonly property bool separator: modelData.id === "separator"

            width: parent.width
            visible: modelData.visible !== false
            height: visible ? (separator ? Theme.spacingXS : Theme.menuItemHeight) : 0

            Rectangle {
                anchors.centerIn: parent
                visible: slot.separator
                width: parent.width - Theme.spacingM * 2
                height: Theme.dividerWidth
                color: Theme.outlineVariant
            }

            ContextMenuItem {
                anchors.fill: parent
                visible: !slot.separator
                text: slot.separator ? "" : slot.modelData.label
                iconName: slot.separator ? "" : slot.modelData.icon
                enabled: !slot.separator && slot.modelData.enabled
                onClicked: {
                    menu.close();
                    menu.actionTriggered(slot.modelData.id);
                }
            }
        }
    }
}
