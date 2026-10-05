pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services
import qs.DCommon.FileBrowser

ShellRoot {
    id: fixture

    readonly property var log: Log.scoped("fixture")
    property int failures: 0
    property var started: []
    property var cancelled: []
    property var startResponse: ({})
    property int nextId: 0

    function check(label, actual, expected) {
        const ok = JSON.stringify(actual) === JSON.stringify(expected);
        if (!ok) {
            failures++;
            log.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
            return;
        }
        log.info(`ok ${label}`);
    }

    function entry(name) {
        return {
            "name": name,
            "path": "/tmp/" + name,
            "isDir": false,
            "size": 1,
            "mtimeMs": 1000,
            "mime": "text/plain",
            "thumbnailable": name.endsWith(".png"),
            "thumbnail": ""
        };
    }

    function names() {
        const out = [];
        for (let i = 0; i < model.entries.count; i++)
            out.push(model.entries.get(i).name);
        return out;
    }

    readonly property QtObject fakeBackend: QtObject {
        property bool connected: true

        signal searchEvent(var data)

        function search(root, query, options, callback) {
            fixture.nextId++;
            fixture.started = fixture.started.concat([
                {
                    "root": root,
                    "query": query,
                    "options": options
                }
            ]);
            callback(Object.assign({
                "searchId": "s" + fixture.nextId,
                "topic": "search:s" + fixture.nextId,
                "limit": 10000
            }, fixture.startResponse));
        }

        function cancelSearch(searchId) {
            fixture.cancelled = fixture.cancelled.concat([searchId]);
        }

        function thumbnails(paths, size, watchId, callback) {
            callback({
                "results": paths.map(path => ({
                            "path": path,
                            "thumbnail": "/cache/" + path.split("/").pop()
                        }))
            });
        }
    }

    readonly property SearchModel model: SearchModel {
        backend: fixture.fakeBackend
    }

    Component.onCompleted: {
        let finished = 0;
        model.finished.connect(() => finished++);

        model.rootPath = "/tmp";
        model.query = "rep";
        model.options = {
            "types": ["documents"]
        };
        model.start();
        check("start sends root, query and options", started[0], {
            "root": "/tmp",
            "query": "rep",
            "options": {
                "types": ["documents"]
            }
        });
        check("active and loading", [model.active, model.loading, model.searchId], [true, true, "s1"]);

        fakeBackend.searchEvent({
            "searchId": "s1",
            "kind": "results",
            "entries": [entry("report.txt"), entry("Reports")],
            "count": 2
        });
        check("results append", names(), ["report.txt", "Reports"]);
        check("count follows the daemon", model.total, 2);

        fakeBackend.searchEvent({
            "searchId": "s9",
            "kind": "results",
            "entries": [entry("other.txt")],
            "count": 1
        });
        check("another search's events are ignored", names(), ["report.txt", "Reports"]);

        fakeBackend.searchEvent({
            "searchId": "s1",
            "kind": "results",
            "entries": [entry("pic.png")],
            "count": 3
        });
        model.requestThumbnails(0, 2);
        check("thumbnails only for thumbnailable entries", [model.entries.get(0).thumbnail, model.entries.get(2).thumbnail], ["", "/cache/pic.png"]);

        fakeBackend.searchEvent({
            "searchId": "s1",
            "kind": "done",
            "count": 3,
            "truncated": true,
            "unreadable": 2
        });
        check("done ends loading", [model.loading, model.truncated, model.unreadable, finished], [false, true, 2, 1]);
        check("done keeps the results", names().length, 3);
        check("a finished search is not cancelled", cancelled, []);

        model.start();
        check("restart clears the list", names(), []);
        fakeBackend.searchEvent({
            "searchId": "s2",
            "kind": "results",
            "entries": [entry("a")],
            "count": 1
        });
        model.cancel();
        check("cancel tells the backend", cancelled, ["s2"]);
        check("cancel keeps what arrived", names(), ["a"]);
        check("cancel ends loading but stays active", [model.loading, model.active], [false, true]);

        fakeBackend.searchEvent({
            "searchId": "s2",
            "kind": "done",
            "count": 1
        });
        check("late events after cancel are dropped", finished, 1);

        model.clear();
        check("clear leaves results mode", [model.active, names()], [false, []]);

        startResponse = {
            "error": "not a directory",
            "code": "ENOTDIR"
        };
        model.start();
        check("a failed start reports the code", [model.errorCode, model.loading], ["ENOTDIR", false]);

        log.info(failures === 0 ? "FIXTURE PASS" : `FIXTURE FAIL (${failures})`);
        Qt.callLater(Qt.quit);
    }
}
