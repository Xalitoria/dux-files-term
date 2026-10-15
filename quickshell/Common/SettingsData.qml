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
    property real blurLayerOutlineOpacity: 0
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
    property alias windowTitleBars: adapter.windowTitleBars
    // "" follows the system language, otherwise a bundled translation code such as "de"
    property alias language: adapter.language
    // "" autodetects from $TERMINAL and the known emulators
    property alias terminal: adapter.terminal
    // "hide" keeps the daemon warm, "quit" stops it with the last window
    property alias closeBehavior: adapter.closeBehavior
    // "current" replaces the active pane, "tab" opens a new tab
    property alias openLocationIn: adapter.openLocationIn
    property alias extractOnOpen: adapter.extractOnOpen
    property alias registerFileManager: adapter.registerFileManager
    property alias sidebarCollapsed: adapter.sidebarCollapsed
    // "list" | "grid", the view a new window starts in
    property alias defaultView: adapter.defaultView
    // name | size | mtime | type
    property alias defaultSort: adapter.defaultSort
    property alias defaultSortDescending: adapter.defaultSortDescending
    property alias showHidden: adapter.showHidden
    property alias foldersFirst: adapter.foldersFirst
    property alias folderCounts: adapter.folderCounts
    property alias confirmTrash: adapter.confirmTrash
    // "double" | "single"
    property alias clickToOpen: adapter.clickToOpen
    // "auto" decides from the device, else "copy" | "move" | "ask"
    property alias dragAction: adapter.dragAction
    property alias thumbnails: adapter.thumbnails
    // 0 means no cap
    property alias thumbnailSizeLimitMb: adapter.thumbnailSizeLimitMb
    property alias networkThumbnails: adapter.networkThumbnails
    property alias showSystemPartitions: adapter.showSystemPartitions
    property alias mountOnClick: adapter.mountOnClick
    property alias searchHidden: adapter.searchHidden
    property alias vimKeys: adapter.vimKeys
    // "jump" moves to the first matching name, "filter" opens the pane filter
    property alias listTyping: adapter.listTyping
    property alias gtkBookmarksOffered: adapter.gtkBookmarksOffered

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
            property bool windowTitleBars: true
            property string language: ""
            property string terminal: ""
            property string closeBehavior: "quit"
            property string openLocationIn: "current"
            property bool extractOnOpen: true
            property bool registerFileManager: true
            property bool sidebarCollapsed: false
            property string defaultView: "list"
            property string defaultSort: "name"
            property bool defaultSortDescending: false
            property bool showHidden: false
            property bool foldersFirst: true
            property bool folderCounts: true
            property bool confirmTrash: false
            property string clickToOpen: "double"
            property string dragAction: "auto"
            property bool thumbnails: true
            property int thumbnailSizeLimitMb: 64
            property bool networkThumbnails: false
            property bool showSystemPartitions: false
            property bool mountOnClick: true
            property bool searchHidden: false
            property bool vimKeys: false
            property string listTyping: "jump"
            property bool gtkBookmarksOffered: false
        }
    }
}
