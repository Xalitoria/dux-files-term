pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services
import qs.DCommon.FileBrowser

ShellRoot {
    id: fixture

    readonly property var log: Log.scoped("fixture")
    property int failures: 0
    property int watches: 0
    property var previewed: []

    function check(label, actual, expected) {
        const ok = JSON.stringify(actual) === JSON.stringify(expected);
        if (!ok) {
            failures++;
            log.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
            return;
        }
        log.info(`ok ${label}`);
    }

    function entry(dir, name, extra) {
        return Object.assign({
            "name": name,
            "displayName": "",
            "path": dir + "/" + name,
            "isDir": false,
            "untrusted": false,
            "isSymlink": false,
            "symlinkTarget": "",
            "symlinkBroken": false,
            "size": 12,
            "mtimeMs": 1000,
            "ctimeMs": 1000,
            "mode": "-rw-r--r--",
            "owner": "me",
            "group": "me",
            "hidden": false,
            "isExecutable": false,
            "extension": "txt",
            "mime": "text/plain",
            "iconName": "text-plain",
            "thumbnail": "",
            "thumbnailable": false,
            "unreadable": false
        }, extra || {});
    }

    function folder(dir, name) {
        return entry(dir, name, {
            "isDir": true,
            "extension": "",
            "mime": "inode/directory",
            "iconName": "folder"
        });
    }

    readonly property var tree: ({
            "/home/me": [folder("/home/me", "docs"), entry("/home/me", "notes.txt")],
            "/home/me/docs": [folder("/home/me/docs", "sub"), entry("/home/me/docs", "readme.txt"), entry("/home/me/docs", "logo.png", {
                    "mime": "image/png",
                    "extension": "png"
                })],
            "/home/me/docs/sub": []
        })

    readonly property QtObject fakeService: QtObject {
        property bool connected: true
        property var userDirs: [
            {
                "key": "home",
                "path": "/home/me",
                "name": "Home",
                "iconName": "user-home"
            }
        ]

        signal watchEvent(var data)

        function watch(path, options, callback) {
            fixture.watches++;
            callback({
                "watchId": "w" + fixture.watches,
                "topic": "files:w" + fixture.watches,
                "watching": true,
                "pollOnFocus": false,
                "path": path,
                "total": (fixture.tree[path] ?? []).length,
                "cursor": "",
                "entries": fixture.tree[path] ?? []
            });
        }

        function page(watchId, options, callback) {
            callback({});
        }

        function unwatch(watchId) {
        }

        function thumbnails(paths, size, watchId, callback) {
            callback({
                "results": []
            });
        }

        function icons(paths, callback) {
            callback({});
        }

        function count(paths, includeHidden, callback) {
            callback({
                "counts": {}
            });
        }

        function preview(path, maxBytes, callback) {
            fixture.previewed.push(path);
            callback({
                "text": "hello from " + path,
                "binary": false,
                "truncated": true,
                "size": 4096
            });
        }
    }

    readonly property Item host: Item {
        width: 1200
        height: 800

        FileBrowserView {
            id: browser

            anchors.fill: parent
            backend: fixture.fakeService
            viewMode: "columns"
            showSidebar: false
            showNavigation: false
        }
    }

    readonly property FilePreviewPane preview: FilePreviewPane {
        width: 300
        height: 600
        backend: fixture.fakeService
    }

    function columns() {
        return browser.body.view;
    }

    Component.onCompleted: {
        browser.navigate("/home/me/docs");
        check("active column lists the current folder", browser.body.directory.count, 3);
        check("ancestors run from home", columns().ancestors, ["/home/me"]);
        check("nothing picked at first", [columns().lookaheadPath, columns().pickedEntry], ["", null]);

        browser.selection.select("/home/me/docs/sub");
        check("a selected folder opens a lookahead column", columns().lookaheadPath, "/home/me/docs/sub");

        browser.selection.select("/home/me/docs/readme.txt");
        check("a selected file closes the lookahead", columns().lookaheadPath, "");
        check("the preview pane is at least its minimum and grows into the free width", [columns().trailingWidth >= FileBrowserMetrics.previewPaneWidth, columns().trailingWidth <= 1200], [true, true]);
        check("a selected file is the preview entry", columns().pickedEntry.name, "readme.txt");

        browser.selection.select("/home/me/docs/sub");
        browser.body.enterCursor();
        check("entering the cursor folder makes it the active column", browser.path, "/home/me/docs/sub");
        check("the chain grows by one", columns().ancestors, ["/home/me", "/home/me/docs"]);

        browser.body.leaveColumn();
        check("leaving goes to the parent", browser.path, "/home/me/docs");
        check("the former folder stays selected", browser.selection.paths, ["/home/me/docs/sub"]);
        check("so its column stays open as the lookahead", columns().lookaheadPath, "/home/me/docs/sub");

        browser.body.revealRequested("/home/me/notes.txt");
        check("revealing an ancestor item moves the active column", browser.path, "/home/me");
        check("and selects the item there", browser.selection.paths, ["/home/me/notes.txt"]);
        check("the chain shrinks", columns().ancestors, []);

        browser.reveal("/home/me/gone.txt");
        check("a reveal that misses drops the old selection", browser.selection.paths, []);

        preview.entry = fixture.tree["/home/me/docs"][1];
        preview.load();
        check("text preview comes from the backend", preview.text, "hello from /home/me/docs/readme.txt");
        check("truncation is reported", preview.truncated, true);
        check("text files are previewable", preview.canPreview, true);

        preview.imageSizeLimit = 1;
        preview.entry = fixture.tree["/home/me/docs"][2];
        preview.load();
        check("images skip the text request", [preview.canPreview, preview.imageLike, fixture.previewed.length], [false, false, 1]);

        browser.navigate("/home/me/docs");
        browser.selection.clear();
        const stray = {
            "key": Qt.Key_Down,
            "modifiers": Qt.NoModifier,
            "text": "",
            "accepted": false
        };
        browser.body.handleKey(stray);
        check("a key handed to the body moves the cursor", [stray.accepted, browser.selection.cursorPath], [true, "/home/me/docs/sub"]);

        browser.navigate("/");
        check("the root chain is empty at /", columns().ancestors, []);
        browser.navigate("/home");
        check("outside home the chain starts at /", columns().ancestors, ["/"]);

        if (failures === 0)
            log.info("FIXTURE PASS");
        Qt.callLater(Qt.quit);
    }
}
