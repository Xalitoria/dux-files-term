pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Item {
    id: root

    property string iconName: ""
    property string thumbnail: ""
    property int size: Theme.iconSize
    property color color: Theme.surfaceText

    readonly property string themeIconPath: Paths.resolveIconPath(iconName)
    readonly property string bundledIcon: _bundled(iconName)

    implicitWidth: size
    implicitHeight: size

    function _bundled(name) {
        switch (name) {
        case "folder":
        case "folder-documents":
        case "folder-download":
        case "folder-music":
        case "folder-pictures":
        case "folder-publicshare":
        case "folder-remote":
        case "folder-templates":
        case "folder-videos":
            return name;
        case "user-home":
        case "user-desktop":
        case "inode-directory":
            return "folder";
        default:
            return "";
        }
    }

    function _glyph(name) {
        switch (true) {
        case name.startsWith("folder"):
        case name.startsWith("user-"):
        case name === "inode-directory":
            return "folder";
        case name === "inode-symlink":
            return "link";
        case name === "emblem-unreadable":
        case name === "lock":
            return "lock";
        case name.startsWith("drive-removable"):
            return "usb";
        case name.startsWith("drive-"):
            return "hard_drive";
        case name === "media-optical":
            return "album";
        case name === "multimedia-player":
        case name === "phone":
            return "smartphone";
        case name === "camera-photo":
            return "photo_camera";
        case name === "application-pdf":
            return "picture_as_pdf";
        case name.startsWith("image"):
            return "image";
        case name.startsWith("video"):
            return "movie";
        case name.startsWith("audio"):
            return "music_note";
        case name.startsWith("text"):
            return "description";
        case name.startsWith("application-x-executable"):
        case name.startsWith("application-x-shellscript"):
            return "terminal";
        case name.startsWith("application-zip"):
        case name.startsWith("application-gzip"):
        case name.startsWith("application-zstd"):
        case name.startsWith("application-x-7z"):
        case name.startsWith("application-x-tar"):
            return "folder_zip";
        default:
            return "draft";
        }
    }

    Image {
        anchors.fill: parent
        visible: root.thumbnail !== ""
        source: root.thumbnail === "" ? "" : Paths.toFileUrl(root.thumbnail)
        sourceSize.width: root.size * 2
        sourceSize.height: root.size * 2
        fillMode: Image.PreserveAspectFit
        asynchronous: true
        cache: true
        smooth: true
    }

    DankSVGIcon {
        anchors.fill: parent
        visible: root.thumbnail === "" && root.themeIconPath !== ""
        source: root.themeIconPath
        size: root.size
    }

    Image {
        anchors.fill: parent
        visible: root.thumbnail === "" && root.themeIconPath === "" && root.bundledIcon !== ""
        source: root.bundledIcon === "" ? "" : Qt.resolvedUrl("../assets/icons/" + root.bundledIcon + ".svg")
        sourceSize.width: root.size * 2
        sourceSize.height: root.size * 2
        fillMode: Image.PreserveAspectFit
        asynchronous: true
        smooth: true
    }

    DankIcon {
        anchors.centerIn: parent
        visible: root.thumbnail === "" && root.themeIconPath === "" && root.bundledIcon === ""
        name: root._glyph(root.iconName)
        size: root.size
        color: root.color
    }
}
