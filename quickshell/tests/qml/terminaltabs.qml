pragma ComponentBehavior: Bound

import QtQuick
import QtTest
import Quickshell
import qs.Common
import qs.Services
import qs.Modules
import qs.DCommon.Common as DC

ShellRoot {
    id: root

    readonly property var log: Log.scoped("fixture")

    property var window: null
    property bool ready: false
    property int lastSession: 0
    property var opened: []
    property var closed: []
    property var followed: []
    property var restyled: []
    property var handlers: ({})

    Component {
        id: mainWindow

        MainWindow {
            windowId: 1
            terminalTransport: root.fakeTransport
        }
    }

    Component.onCompleted: {
        DC.Style.theme = Theme;
        DC.Style.settings = SettingsData;
        DC.I18n.backend = I18n;
        DC.Paths.backend = Paths;
        DC.Log.backend = Log;
        DC.Host.session = SessionService;
        DC.Host.cache = CacheData;
        DC.Host.files = DirectoryService;
        window = mainWindow.createObject(root);
        ready = true;
    }

    readonly property QtObject fakeTransport: QtObject {
        function subscribeTopic(topic, handler) {
            root.handlers[topic] = handler;
        }
        function unsubscribeTopic(topic, handler) {
            if (root.handlers[topic] === handler)
                delete root.handlers[topic];
        }
        function call(method, params, callback) {
            switch (method) {
            case "terminal.open":
                const id = String(++root.lastSession);
                root.opened = root.opened.concat([
                    {
                        "id": id,
                        "cwd": params.cwd
                    }
                ]);
                callback?.({
                    "id": id,
                    "topic": "terminal:" + id,
                    "follow": true
                });
                return;
            case "terminal.close":
                root.closed = root.closed.concat([params.id]);
                break;
            case "terminal.restyle":
                root.restyled = root.restyled.concat([[params.id, params.terminal]]);
                break;
            case "terminal.cd":
                root.followed = root.followed.concat([[params.id, params.path]]);
                break;
            }
            callback?.({});
        }
    }

    TestCase {
        name: "TerminalTabs"
        when: root.ready

        function views(item, found) {
            if (item instanceof TerminalView)
                found.push(item);
            for (const child of item.children)
                views(child, found);
            return found;
        }

        function shown() {
            return views(root.window.contentItem, []).filter(view => view.visible && view.sessionId !== "").map(view => view.sessionId);
        }

        function terminalCount() {
            return views(root.window.contentItem, []).length;
        }

        function openedIds() {
            return root.opened.map(entry => entry.id);
        }

        function exitShell(id) {
            root.handlers["terminal:" + id]({
                "exited": true
            });
        }

        function test_terminalTabs() {
            scenario_terminalPerTab();
            scenario_pickingATerminalRestylesOpenTerminals();
            scenario_backgroundTabsAndDuplicates();
            scenario_closingTabsClosesTheirTerminals();
            scenario_closingATabBesideTheCurrentOneStartsNothing();
            scenario_hiddenPanelStartsNothingUntilToggled();
            scenario_reopenedTabGetsAFreshShell();
            scenario_exitingAShellOnlyEndsItsOwnTab();
            scenario_closingTheWindowEndsEveryShell();
            root.log.info("FIXTURE PASS");
        }

        function scenario_terminalPerTab() {
            const win = root.window;
            const tabs = win.tabManager;
            win.minimumSize = Qt.size(1100, 760);
            win.implicitWidth = 1100;
            win.implicitHeight = 760;
            SettingsData.animationDuration = 0;
            const surface = win.contentItem.children.find(child => child.isFloatingWindowSurface === true);
            tryCompare(surface, "width", 1100);

            compare(tabs.tabs.length, 1);
            compare(tabs.currentTabId, tabs.tabs[0].id);
            compare(root.opened.length, 0);

            win.activePane.navigate("/tmp");
            compare(tabs.tabPath(tabs.currentTabId), "/tmp");
            tabs.newTab("/usr", false);
            tabs.selectTab(0);
            compare(root.opened.length, 0);
            compare(win.terminalOpen, false);

            win.setTerminalMode("bottom");
            compare(openedIds(), ["1"]);
            compare(root.opened[0].cwd, "/tmp");
            compare(shown(), ["1"]);
            verify(win.activeTerminal !== null);
            compare(win.activeTerminal.sessionId, "1");

            tabs.selectTab(1);
            compare(openedIds(), ["1", "2"]);
            compare(root.opened[1].cwd, "/usr");
            compare(shown(), ["2"]);
            compare(win.activeTerminal.sessionId, "2");
            compare(root.followed, []);

            tabs.selectTab(0);
            compare(openedIds(), ["1", "2"]);
            compare(shown(), ["1"]);
            compare(win.activeTerminal.sessionId, "1");
            compare(root.closed, []);
            compare(root.followed, []);

            win.activePane.navigate("/var");
            compare(root.followed, [["1", "/var"]]);
            tabs.selectTab(1);
            win.activePane.navigate("/opt");
            compare(root.followed, [["1", "/var"], ["2", "/opt"]]);
            tabs.selectTab(0);
            compare(root.followed.length, 2);
        }

        function scenario_pickingATerminalRestylesOpenTerminals() {
            const live = views(root.window.contentItem, []).filter(view => view.sessionId !== "");
            verify(live.length > 0);
            root.restyled = [];

            SettingsData.terminal = "ghostty";
            compare(root.restyled.length, live.length);
            for (const view of live)
                verify(root.restyled.some(entry => entry[0] === view.sessionId && entry[1] === "ghostty"));

            const first = live[0];
            root.handlers["terminal:" + first.sessionId]({
                "theme": {
                    "colors": {
                        "bg": "#0f0028"
                    },
                    "behavior": {
                        "copyOnSelect": true
                    }
                }
            });
            compare(first.colors.bg, "#0f0028");
            compare(first.behavior.copyOnSelect, true);

            SettingsData.terminal = "";
        }

        function scenario_backgroundTabsAndDuplicates() {
            const win = root.window;
            const tabs = win.tabManager;
            const before = root.opened.length;

            tabs.newTab("/srv", true);
            compare(root.opened.length, before);
            compare(shown(), ["1"]);

            tabs.selectTab(2);
            compare(root.opened.length, before + 1);
            compare(root.opened[before].cwd, "/srv");
            compare(shown(), [root.opened[before].id]);

            tabs.duplicateTab(2);
            compare(tabs.currentTab, 3);
            compare(root.opened.length, before + 2);
            compare(root.opened[before + 1].cwd, "/srv");
            verify(root.opened[before].id !== root.opened[before + 1].id);
            compare(shown(), [root.opened[before + 1].id]);
            verify(tabs.tabs[2].id !== tabs.tabs[3].id);
        }

        function scenario_closingTabsClosesTheirTerminals() {
            const win = root.window;
            const tabs = win.tabManager;
            const total = tabs.tabs.length;
            const sessions = root.opened.length;
            const closedBefore = root.closed.length;
            const last = root.opened[sessions - 1].id;

            tabs.closeTab(0);
            compare(tabs.tabs.length, total - 1);
            compare(root.closed.slice(closedBefore), ["1"]);
            compare(root.opened.length, sessions);
            compare(shown(), [last]);
            compare(win.activeTerminal.sessionId, last);

            const neighbour = root.opened[sessions - 2].id;
            tabs.closeTab(tabs.currentTab);
            compare(root.closed.length, closedBefore + 2);
            compare(root.closed[closedBefore + 1], last);
            compare(root.opened.length, sessions);
            compare(win.activeTerminal.sessionId, neighbour);
            compare(shown(), [neighbour]);
        }

        function scenario_closingATabBesideTheCurrentOneStartsNothing() {
            const win = root.window;
            const tabs = win.tabManager;
            win.setTerminalMode("hidden");
            while (tabs.tabs.length > 1)
                tabs.closeTab(tabs.tabs.length - 1);
            tabs.newTab("/a", false);
            tabs.newTab("/b", false);
            tabs.newTab("/c", false);
            win.setTerminalMode("bottom");
            const sessions = root.opened.length;
            const current = win.activeTerminal.sessionId;
            const closedBefore = root.closed.length;

            tabs.closeTab(1);
            tabs.closeTab(1);
            compare(root.opened.length, sessions);
            compare(root.closed.length, closedBefore);
            compare(win.activeTerminal.sessionId, current);
            compare(shown(), [current]);
            compare(tabs.tabs.length, 2);
        }

        function scenario_hiddenPanelStartsNothingUntilToggled() {
            const win = root.window;
            const tabs = win.tabManager;
            win.setTerminalMode("hidden");
            const sessions = root.opened.length;
            tabs.selectTab(0);
            tabs.newTab("/hidden", false);
            compare(root.opened.length, sessions);
            compare(win.activeTerminal, null);
            compare(shown(), []);

            win.setTerminalMode("files");
            compare(root.opened.length, sessions + 1);
            compare(root.opened[sessions].cwd, "/hidden");
            compare(shown(), [root.opened[sessions].id]);
            win.setTerminalMode("hidden");
            win.setTerminalMode("files");
            compare(root.opened.length, sessions + 1);
        }

        function scenario_reopenedTabGetsAFreshShell() {
            const win = root.window;
            const tabs = win.tabManager;
            win.setTerminalMode("bottom");
            tabs.newTab("/reopen", false);
            const doomed = win.activeTerminal.sessionId;
            tabs.closeTab(tabs.currentTab);
            verify(root.closed.includes(doomed));
            const sessions = root.opened.length;

            tabs.reopenTab();
            compare(root.opened.length, sessions + 1);
            compare(root.opened[sessions].cwd, "/reopen");
            verify(root.opened[sessions].id !== doomed);
            compare(shown(), [root.opened[sessions].id]);
        }

        function scenario_exitingAShellOnlyEndsItsOwnTab() {
            const win = root.window;
            const tabs = win.tabManager;
            win.setTerminalMode("bottom");
            tabs.selectTab(0);
            const first = win.activeTerminal.sessionId;
            tabs.selectTab(1);
            const second = win.activeTerminal.sessionId;
            verify(first !== second);

            const count = terminalCount();

            exitShell(first);
            tryVerify(() => terminalCount() === count - 1);
            compare(win.terminalOpen, true);
            compare(shown(), [second]);
            compare(win.activeTerminal.sessionId, second);

            exitShell(second);
            compare(win.terminalOpen, false);
            tryVerify(() => terminalCount() === count - 2);
            compare(win.activeTerminal, null);

            const sessions = root.opened.length;
            win.setTerminalMode("bottom");
            compare(root.opened.length, sessions + 1);
            compare(shown(), [root.opened[sessions].id]);
        }

        function scenario_closingTheWindowEndsEveryShell() {
            const win = root.window;
            const tabs = win.tabManager;
            win.setTerminalMode("bottom");
            for (let index = 0; index < tabs.tabs.length; index++)
                tabs.selectTab(index);
            const running = views(win.contentItem, []).map(view => view.sessionId);
            verify(running.length >= 2);
            verify(running.every(id => id !== ""));

            win.destroy();
            root.window = null;
            tryVerify(() => running.every(id => root.closed.includes(id)));
        }
    }
}
