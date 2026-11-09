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
    property var typed: []
    property var searched: []
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
            case "terminal.input":
                root.typed = root.typed.concat([params]);
                break;
            case "terminal.restyle":
                root.restyled = root.restyled.concat([[params.id, params.terminal]]);
                break;
            case "terminal.search":
                root.searched = root.searched.concat([params]);
                callback?.({
                    "count": params.query === "" ? 0 : 2,
                    "index": params.query === "" ? 0 : 1
                });
                return;
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
            scenario_rightTerminalResizes();
            scenario_rightAndMiddleClicks();
            scenario_scrollbackSearch();
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

        function scenario_rightTerminalResizes() {
            const win = root.window;
            const width = win.contentItem.children.find(child => child.isFloatingWindowSurface === true).width;
            const mode = win.terminalMode;

            win.setTerminalMode("right");
            compare(win.terminalRightStart, width / 2);
            compare(win.terminalRightInset, width / 2 + FilesMetrics.splitterWidth);

            win.setTerminalStart(width * 0.4);
            compare(win.terminalRightStart, width * 0.4);
            compare(win.terminalRightInset, width * 0.6 + FilesMetrics.splitterWidth);
            fuzzyCompare(SessionData.terminalWidthRatio, 0.6, 0.001);

            win.setTerminalStart(0);
            compare(win.terminalRightStart, width * FilesMetrics.minimumPaneFraction);
            win.setTerminalStart(width);
            compare(win.terminalRightStart, width - FilesMetrics.terminalMinWidth);

            win.setTerminalStart(width / 2);
            win.setTerminalMode(mode);
        }

        function scenario_rightAndMiddleClicks() {
            const view = root.window.activeTerminal;
            verify(view !== null);
            verify(view.sessionId !== "");
            compare(view.rightClick, "menu");
            compare(view.middleClick, "primaryPaste");

            view.behavior = {
                "rightClick": "extend",
                "middleClick": "paste"
            };
            compare(view.rightClick, "extend");
            compare(view.middleClick, "paste");

            root.typed = [];
            view.primaryText = "picked";
            view.clickAction("primaryPaste", Qt.point(0, 0), Qt.point(0, 0));
            compare(root.typed.length, 1);
            compare(root.typed[0].text, "picked");
            compare(root.typed[0].paste, true);
            view.primaryText = "";
            view.clickAction("primaryPaste", Qt.point(0, 0), Qt.point(0, 0));
            compare(root.typed.length, 1, "nothing selected, nothing pasted");

            view.cols = 20;
            view.rows = 5;
            view.select(Qt.point(2, 1), Qt.point(5, 1));
            view.extendSelection(Qt.point(9, 1));
            compare(view.selStart.x, 2);
            compare(view.selEnd.x, 9);
            view.extendSelection(Qt.point(0, 1));
            compare(view.selStart.x, 0);
            compare(view.selEnd.x, 9);

            view.clearSelection();
            view.extendSelection(Qt.point(3, 1));
            compare(view.selStart, null, "no selection to extend");

            view.selectAll();
            compare(view.selStart, Qt.point(0, 0));
            compare(view.selEnd, Qt.point(19, 4));

            view.clearSelection();
            view.behavior = {};
            compare(view.rightClick, "menu");
        }

        function scenario_scrollbackSearch() {
            const view = root.window.activeTerminal;
            verify(view !== null);
            verify(view.sessionId !== "");
            compare(view.searching, false);

            view.behavior = {
                "keys": {
                    "search": ["ctrl+shift+r"]
                }
            };
            compare(view.actionFor("ctrl+shift+r"), "search", "the terminal's own key opens the search");
            compare(view.actionFor("ctrl+shift+f"), "search");
            view.behavior = {};

            root.searched = [];
            verify(view.runAction("search"));
            compare(view.searching, true);
            compare(root.searched.length, 0, "an empty query asks nothing");

            view.searchQuery = "foo";
            compare(root.searched.length, 1);
            compare(root.searched[0].id, view.sessionId);
            compare(root.searched[0].query, "foo");
            compare(root.searched[0].step, 0);
            compare(view.matchCount, 2);
            compare(view.matchIndex, 1);

            view.runSearch(-1);
            compare(root.searched[1].step, -1);
            view.runSearch(1);
            compare(root.searched[2].step, 1);

            root.handlers["terminal:" + view.sessionId]({
                "cols": 20,
                "rows": 5,
                "scroll": 0,
                "lines": [],
                "cursor": {
                    "x": 0,
                    "y": 0,
                    "visible": false
                },
                "matches": [
                    {
                        "y": 1,
                        "x": 2,
                        "w": 3,
                        "current": true
                    },
                    {
                        "y": 3,
                        "x": 0,
                        "w": 3
                    }
                ]
            });
            compare(view.matches.length, 2);

            view.closeSearch();
            compare(view.searching, false);
            compare(root.searched.length, 4);
            compare(root.searched[3].query, "", "closing ends the search in the core");
            compare(view.matches.length, 0);
            compare(view.matchCount, 0);

            view.searchQuery = "";
            compare(root.searched.length, 4, "typing into a closed search asks nothing");
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
