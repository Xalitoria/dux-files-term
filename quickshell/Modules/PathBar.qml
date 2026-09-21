pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

FocusScope {
    id: bar

    property string path: ""
    property string homePath: Paths.strip(Paths.home)
    property bool editing: false

    signal navigated(string target)

    readonly property var segments: bar.segmentsFor(path, homePath)

    function segmentsFor(path, home) {
        if (path === "")
            return [];

        const rooted = home !== "" && (path === home || path.startsWith(home + "/"));
        const head = rooted ? {
            "label": "",
            "iconName": "home",
            "path": home
        } : {
            "label": "",
            "iconName": "hard_drive",
            "path": "/"
        };

        const rest = rooted ? path.substring(home.length) : path;
        const out = [head];
        let walked = head.path === "/" ? "" : head.path;
        for (const name of rest.split("/")) {
            if (name === "")
                continue;
            walked = walked + "/" + name;
            out.push({
                "label": name,
                "iconName": "",
                "path": walked
            });
        }
        return out;
    }

    function startEditing() {
        editing = true;
        field.text = path;
        field.forceActiveFocus();
        field.selectAll();
    }

    function stopEditing() {
        editing = false;
        forceActiveFocus();
    }

    implicitHeight: FilesMetrics.pathPillHeight

    Keys.onEscapePressed: stopEditing()

    DankFlickable {
        id: strip

        anchors.fill: parent
        visible: !bar.editing
        wheelEnabled: false
        clip: true
        flickableDirection: Flickable.HorizontalFlick
        contentWidth: pills.width
        contentHeight: height

        onContentWidthChanged: contentX = Math.max(0, contentWidth - width)
        onWidthChanged: contentX = Math.max(0, contentWidth - width)

        WheelHandler {
            acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
            onWheel: event => {
                const delta = event.angleDelta.y !== 0 ? event.angleDelta.y : event.angleDelta.x;
                strip.contentX = Math.max(0, Math.min(strip.contentWidth - strip.width, strip.contentX - delta));
            }
        }

        MouseArea {
            width: Math.max(strip.width, pills.width)
            height: strip.height
            onClicked: bar.startEditing()
        }

        Row {
            id: pills

            height: strip.height
            spacing: FilesMetrics.pathBarSpacing

            Repeater {
                model: bar.segments

                Row {
                    id: segment

                    required property int index
                    required property var modelData

                    anchors.verticalCenter: parent.verticalCenter
                    spacing: FilesMetrics.pathBarSpacing

                    StyledText {
                        anchors.verticalCenter: parent.verticalCenter
                        visible: segment.index > 0
                        text: "/"
                        color: Theme.outline
                        font.pixelSize: Theme.fontSizeMedium
                    }

                    PathPill {
                        label: segment.modelData.label
                        iconName: segment.modelData.iconName
                        current: segment.index === bar.segments.length - 1
                        onClicked: bar.navigated(segment.modelData.path)
                    }
                }
            }
        }
    }

    DankTextField {
        id: field

        anchors.fill: parent
        visible: bar.editing
        cornerRadius: FilesMetrics.pathPillRadius
        controlHeight: bar.height
        leftIconName: "edit_location"
        leftIconSize: FilesMetrics.pathIconSize
        font.pixelSize: Theme.fontSizeSmall
        keyForwardTargets: [bar]
        onAccepted: {
            const target = Paths.expandTilde(text.trim());
            bar.stopEditing();
            if (target.startsWith("/"))
                bar.navigated(target);
        }
        onFocusStateChanged: hasFocus => {
            if (!hasFocus && bar.editing)
                bar.stopEditing();
        }
    }
}
