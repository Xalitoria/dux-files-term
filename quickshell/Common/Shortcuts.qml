pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    // scope decides who binds the keys: "window", "path" and "selection" become
    // Shortcut objects on the window, "view" keys are handled by the focused
    // file view so they never fire while a text field has focus.
    readonly property var entries: [
        {
            "id": "open",
            "keys": ["Return", "Enter"],
            "group": "navigation",
            "scope": "selection",
            "label": I18n.tr("Open", "context menu entry opening a file")
        },
        {
            "id": "openContainingFolder",
            "keys": ["Ctrl+Return"],
            "group": "navigation",
            "scope": "selection",
            "label": I18n.tr("Open the containing folder", "keyboard shortcut")
        },
        {
            "id": "openWith",
            "keys": ["Ctrl+Shift+O"],
            "group": "navigation",
            "scope": "selection",
            "label": I18n.tr("Open with", "open with dialog header")
        },
        {
            "id": "openTerminal",
            "keys": ["Ctrl+Alt+T"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Open in terminal", "context menu entry opening a terminal")
        },
        {
            "id": "moveCursor",
            "keys": ["Up", "Down", "Left", "Right"],
            "group": "navigation",
            "scope": "view",
            "label": I18n.tr("Move through the folder", "keyboard shortcut")
        },
        {
            "id": "cursorEdge",
            "keys": ["Home", "End", "PgUp", "PgDown"],
            "group": "navigation",
            "scope": "view",
            "label": I18n.tr("Jump to the start, the end or the next screen", "keyboard shortcut")
        },
        {
            "id": "typeAhead",
            "keys": ["A-Z"],
            "group": "navigation",
            "scope": "view",
            "label": I18n.tr("Jump to the first name starting with what you type", "keyboard shortcut")
        },
        {
            "id": "up",
            "keys": ["Alt+Up", "Backspace"],
            "group": "navigation",
            "scope": "view",
            "label": I18n.tr("Go to the parent folder", "keyboard shortcut")
        },
        {
            "id": "back",
            "keys": ["Alt+Left"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Back", "tooltip on the button going to the previous location")
        },
        {
            "id": "forward",
            "keys": ["Alt+Right"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Forward", "tooltip on the button going to the next location")
        },
        {
            "id": "editPath",
            "keys": ["Ctrl+L"],
            "group": "navigation",
            "scope": "path",
            "label": I18n.tr("Type a location", "keyboard shortcut")
        },
        {
            "id": "search",
            "keys": ["Ctrl+F"],
            "group": "navigation",
            "scope": "path",
            "label": I18n.tr("Search in Files", "placeholder in the search field")
        },
        {
            "id": "newTab",
            "keys": ["Ctrl+T"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("New tab", "context menu entry opening a folder in a new tab")
        },
        {
            "id": "closeTab",
            "keys": ["Ctrl+W"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Close tab", "tooltip on the button closing a tab")
        },
        {
            "id": "nextTab",
            "keys": ["Ctrl+Tab"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Next tab", "keyboard shortcut")
        },
        {
            "id": "previousTab",
            "keys": ["Ctrl+Shift+Tab"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Previous tab", "keyboard shortcut")
        },
        {
            "id": "swapPane",
            "keys": ["F6"],
            "group": "navigation",
            "scope": "window",
            "label": I18n.tr("Switch pane", "keyboard shortcut")
        },
        {
            "id": "listView",
            "keys": ["Ctrl+1"],
            "group": "view",
            "scope": "window",
            "label": I18n.tr("List view", "view mode showing one row per item")
        },
        {
            "id": "gridView",
            "keys": ["Ctrl+2"],
            "group": "view",
            "scope": "window",
            "label": I18n.tr("Grid view", "view mode showing items as tiles")
        },
        {
            "id": "zoomIn",
            "keys": ["Ctrl++", "Ctrl+="],
            "group": "view",
            "scope": "window",
            "label": I18n.tr("Larger icons", "keyboard shortcut")
        },
        {
            "id": "zoomOut",
            "keys": ["Ctrl+-"],
            "group": "view",
            "scope": "window",
            "label": I18n.tr("Smaller icons", "keyboard shortcut")
        },
        {
            "id": "zoomWheel",
            "keys": ["Ctrl+Wheel"],
            "group": "view",
            "scope": "view",
            "label": I18n.tr("Resize icons", "keyboard shortcut")
        },
        {
            "id": "showHidden",
            "keys": ["Ctrl+H"],
            "group": "view",
            "scope": "window",
            "label": I18n.tr("Show hidden files", "menu entry listing dot files")
        },
        {
            "id": "reload",
            "keys": ["F5"],
            "group": "view",
            "scope": "window",
            "label": I18n.tr("Reload", "menu entry re-reading the current folder")
        },
        {
            "id": "shortcuts",
            "keys": ["F1"],
            "group": "view",
            "scope": "path",
            "label": I18n.tr("Keyboard shortcuts", "overlay listing every keyboard shortcut")
        },
        {
            "id": "settings",
            "keys": ["Ctrl+,"],
            "group": "view",
            "scope": "path",
            "label": I18n.tr("Settings", "settings window header title")
        },
        {
            "id": "selectAll",
            "keys": ["Ctrl+A"],
            "group": "selection",
            "scope": "window",
            "label": I18n.tr("Select all", "menu entry selecting every item in the folder")
        },
        {
            "id": "invertSelection",
            "keys": ["Ctrl+Shift+A"],
            "group": "selection",
            "scope": "window",
            "label": I18n.tr("Invert the selection", "keyboard shortcut")
        },
        {
            "id": "toggleItem",
            "keys": ["Space"],
            "group": "selection",
            "scope": "view",
            "label": I18n.tr("Add or remove the item under the cursor", "keyboard shortcut")
        },
        {
            "id": "extendSelection",
            "keys": ["Shift+Click", "Shift+Arrows"],
            "group": "selection",
            "scope": "view",
            "label": I18n.tr("Select a range", "keyboard shortcut")
        },
        {
            "id": "toggleWithPointer",
            "keys": ["Ctrl+Click"],
            "group": "selection",
            "scope": "view",
            "label": I18n.tr("Add or remove one item", "keyboard shortcut")
        },
        {
            "id": "clearSelection",
            "keys": ["Esc"],
            "group": "selection",
            "scope": "view",
            "label": I18n.tr("Clear the selection", "keyboard shortcut")
        },
        {
            "id": "copy",
            "keys": ["Ctrl+C"],
            "group": "operations",
            "scope": "selection",
            "label": I18n.tr("Copy", "context menu entry copying the selection")
        },
        {
            "id": "cut",
            "keys": ["Ctrl+X"],
            "group": "operations",
            "scope": "selection",
            "label": I18n.tr("Cut", "context menu entry marking the selection for a move")
        },
        {
            "id": "paste",
            "keys": ["Ctrl+V"],
            "group": "operations",
            "scope": "window",
            "label": I18n.tr("Paste", "context menu entry pasting the clipboard here")
        },
        {
            "id": "pasteLink",
            "keys": ["Ctrl+Shift+V"],
            "group": "operations",
            "scope": "window",
            "label": I18n.tr("Paste as link", "context menu entry creating symlinks from the clipboard")
        },
        {
            "id": "rename",
            "keys": ["F2"],
            "group": "operations",
            "scope": "selection",
            "label": I18n.tr("Rename", "rename dialog header")
        },
        {
            "id": "trash",
            "keys": ["Del"],
            "group": "operations",
            "scope": "selection",
            "label": I18n.tr("Move to Trash", "context menu entry trashing the selection")
        },
        {
            "id": "deletePermanently",
            "keys": ["Shift+Del"],
            "group": "operations",
            "scope": "selection",
            "label": I18n.tr("Delete permanently", "context menu entry deleting the selection for good")
        },
        {
            "id": "undo",
            "keys": ["Ctrl+Z"],
            "group": "operations",
            "scope": "window",
            "label": I18n.tr("Undo", "toast action reversing the last operation")
        },
        {
            "id": "newFolder",
            "keys": ["Ctrl+Shift+N"],
            "group": "operations",
            "scope": "window",
            "label": I18n.tr("New folder", "new folder dialog header")
        },
        {
            "id": "properties",
            "keys": ["Ctrl+I", "Alt+Return"],
            "group": "operations",
            "scope": "window",
            "label": I18n.tr("Properties", "menu entry showing details about the current folder")
        },
        {
            "id": "contextMenu",
            "keys": ["Menu", "Shift+F10"],
            "group": "operations",
            "scope": "window",
            "label": I18n.tr("Open the context menu", "keyboard shortcut")
        }
    ]

    readonly property var groups: [
        {
            "id": "navigation",
            "label": I18n.tr("Getting around", "keyboard shortcut group")
        },
        {
            "id": "view",
            "label": I18n.tr("View", "keyboard shortcut group")
        },
        {
            "id": "selection",
            "label": I18n.tr("Selection", "keyboard shortcut group")
        },
        {
            "id": "operations",
            "label": I18n.tr("Files", "keyboard shortcut group")
        }
    ]

    function bound(scope) {
        return entries.filter(entry => entry.scope === scope);
    }

    function inGroup(group) {
        return entries.filter(entry => entry.group === group);
    }
}
