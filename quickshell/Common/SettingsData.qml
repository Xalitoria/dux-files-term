pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.Common

Singleton {
    id: root

    enum AnimationSpeed {
        None,
        Short,
        Medium,
        Long,
        Custom
    }

    enum TextRenderType {
        Qt,
        Native,
        Curve
    }

    enum TextRenderQuality {
        Default,
        Low,
        Normal,
        High,
        VeryHigh
    }

    property int animationSpeed: SettingsData.AnimationSpeed.Short
    property bool popoutElevationEnabled: true
    property int textRenderType: SettingsData.TextRenderType.Qt
    property int textRenderQuality: SettingsData.TextRenderQuality.Default
    property bool blurBorderEnabled: true
    property real blurBorderOpacity: 0.35
    property string blurBorderColor: "outline"
    property string blurBorderCustomColor: "#ffffff"
    property bool powerActionConfirm: true
    property real powerActionHoldDuration: 0.5
    property var powerMenuActions: ["reboot", "logout", "poweroff", "lock", "suspend", "restart"]
    property string powerMenuDefaultAction: "logout"
    property bool powerMenuGridLayout: false

    readonly property string configDir: {
        const xdg = Quickshell.env("XDG_CONFIG_HOME");
        const base = (xdg && xdg !== "") ? xdg : Quickshell.env("HOME") + "/.config";
        return base + "/dfiles";
    }
    readonly property string settingsPath: configDir + "/ui-settings.json"

    // "auto" | "light" | "dark"
    property alias themeMode: adapter.themeMode
    // "auto" | "preset" | "custom"
    property alias colorSource: adapter.colorSource
    property alias presetTheme: adapter.presetTheme
    property alias customThemeFile: adapter.customThemeFile
    // 0 (square) … 100; 50 keeps the Material shape scale
    property alias radiusStrength: adapter.radiusStrength
    // base transition length in ms, 0 disables animations
    property alias animationDuration: adapter.animationDuration
    property alias reduceMotion: adapter.reduceMotion
    // 0 smooth, 1 balanced, 2 playful
    property alias springBounce: adapter.springBounce
    property alias enableRippleEffects: adapter.enableRippleEffects
    property alias fontWeight: adapter.fontWeight
    property alias fontScale: adapter.fontScale
    property alias focusRingEnabled: adapter.focusRingEnabled
    property alias focusRingWidth: adapter.focusRingWidth
    // "primary" | "secondary" | "outline" | "surfaceText"
    property alias focusRingColor: adapter.focusRingColor
    // "" follows the system language, otherwise a bundled translation code such as "de"
    property alias language: adapter.language

    readonly property var locale: language !== "" ? Qt.locale(language) : Qt.locale()

    Component.onCompleted: Paths.mkdir(configDir)

    FileView {
        id: settingsFile
        path: root.settingsPath
        blockLoading: true
        blockWrites: true
        atomicWrites: true
        watchChanges: true
        printErrors: false

        property bool _loaded: false
        property bool _selfWrite: false

        onLoaded: _loaded = true
        onLoadFailed: _loaded = true
        onFileChanged: {
            if (_selfWrite) {
                _selfWrite = false;
                return;
            }
            reload();
        }
        onAdapterUpdated: {
            if (!_loaded)
                return;
            _selfWrite = true;
            writeAdapter();
        }

        JsonAdapter {
            id: adapter
            property string themeMode: "auto"
            property string colorSource: "auto"
            property string presetTheme: "purple"
            property string customThemeFile: ""
            property int radiusStrength: 50
            property int animationDuration: 250
            property bool reduceMotion: false
            property int springBounce: 1
            property bool enableRippleEffects: true
            property int fontWeight: 400
            property real fontScale: 1.0
            property bool focusRingEnabled: true
            property real focusRingWidth: 1.5
            property string focusRingColor: "primary"
            property string language: ""
        }
    }
}
