pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.Models
import qs.Services

ShellRoot {
    id: fixture

    readonly property var log: Log.scoped("fixture")
    property int failures: 0
    property var pageResponse: ({})
    property var watchResponse: ({})
    property var eventHandler: null
    property var unwatched: []

    function check(label, actual, expected) {
        const ok = JSON.stringify(actual) === JSON.stringify(expected);
        if (!ok) {
            failures++;
            log.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
            return;
        }
        log.info(`ok ${label}`);
    }

    function entry(name, extra) {
        return Object.assign({
            "name": name,
            "path": "/tmp/" + name,
            "isDir": false,
            "isSymlink": false,
            "symlinkTarget": "",
            "symlinkBroken": false,
            "size": 1,
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

    function names() {
        const out = [];
        for (let i = 0; i < model.entries.count; i++)
            out.push(model.entries.get(i).name);
        return out;
    }

    readonly property QtObject fakeService: QtObject {
        property bool connected: true

        function watch(path, options, callback) {
            callback(fixture.watchResponse);
        }

        function page(watchId, options, callback) {
            callback(fixture.pageResponse);
        }

        function unwatch(watchId) {
            fixture.unwatched.push(watchId);
        }

        function subscribe(topic, handler) {
            fixture.eventHandler = handler;
        }

        function unsubscribe(topic, handler) {
            fixture.eventHandler = null;
        }

        function thumbnails(paths, size, watchId, callback) {
            callback({
                "results": paths.map(path => ({
                            "path": path,
                            "thumbnail": "/cache/" + path.split("/").pop() + ".png"
                        }))
            });
        }

        function icons(paths, callback) {
            callback({});
        }
    }

    readonly property Process finish: Process {
        id: finish
        command: ["sh", "-c", "kill $PPID"]
    }

    readonly property DirectoryModel model: DirectoryModel {
        service: fixture.fakeService
    }

    Component.onCompleted: {
        watchResponse = {
            "watchId": "w1",
            "topic": "files:w1",
            "watching": true,
            "pollOnFocus": false,
            "path": "/tmp",
            "total": 3,
            "cursor": "",
            "entries": [entry("a.txt"), entry("c.txt"), entry("d.txt")]
        };
        model.path = "/tmp";

        check("snapshot entries", names(), ["a.txt", "c.txt", "d.txt"]);
        check("snapshot total", model.total, 3);
        check("watch id", model.watchId, "w1");
        check("subscribed", eventHandler !== null, true);

        eventHandler({
            "watchId": "w1",
            "kind": "batch",
            "seq": 1,
            "added": [entry("b.txt")],
            "addedAt": [1]
        });
        check("insert at index", names(), ["a.txt", "b.txt", "c.txt", "d.txt"]);
        check("total after insert", model.total, 4);

        eventHandler({
            "watchId": "w1",
            "kind": "batch",
            "seq": 2,
            "removed": ["c.txt"],
            "changed": [entry("a.txt", {
                    "size": 42
                })]
        });
        check("remove by name", names(), ["a.txt", "b.txt", "d.txt"]);
        check("changed entry", model.entries.get(0).size, 42);
        check("total after remove", model.total, 3);

        eventHandler({
            "watchId": "w1",
            "kind": "thumbnails",
            "seq": 3,
            "thumbnails": [entry("b.txt", {
                    "thumbnail": "/cache/b.png"
                })]
        });
        check("thumbnail applied", model.entries.get(1).thumbnail, "/cache/b.png");

        eventHandler({
            "watchId": "w2",
            "kind": "batch",
            "seq": 4,
            "removed": ["a.txt"]
        });
        check("other watch ignored", names(), ["a.txt", "b.txt", "d.txt"]);

        pageResponse = {
            "path": "/tmp",
            "total": 1,
            "cursor": "",
            "seq": 9,
            "entries": [entry("resynced.txt")]
        };
        eventHandler({
            "watchId": "w1",
            "kind": "batch",
            "seq": 9,
            "removed": ["b.txt"]
        });
        check("seq gap re-lists", names(), ["resynced.txt"]);

        eventHandler({
            "watchId": "w1",
            "kind": "resync",
            "seq": 10
        });
        check("resync re-lists", names(), ["resynced.txt"]);

        model.requestThumbnails(0, 0);
        check("thumbnail request skips non-thumbnailable", model.entries.get(0).thumbnail, "");

        pageResponse = {
            "total": 1,
            "cursor": "",
            "seq": 10,
            "entries": [entry("pic.png", {
                    "thumbnailable": true,
                    "mime": "image/png"
                })]
        };
        model.setSort("size", true);
        check("sort re-pages", names(), ["pic.png"]);
        model.requestThumbnails(0, 0);
        check("thumbnail request fills cache path", model.entries.get(0).thumbnail, "/cache/pic.png.png");

        eventHandler({
            "watchId": "w1",
            "kind": "batch",
            "seq": 11,
            "changed": [entry("pic.png", {
                    "thumbnailable": true,
                    "mime": "image/png"
                })]
        });
        check("late thumbnailable schedules a retry", model._thumbnailRetry.running, true);

        let goneSeen = false;
        model.gone.connect(() => goneSeen = true);
        eventHandler({
            "watchId": "w1",
            "kind": "gone",
            "seq": 12
        });
        check("gone signal", goneSeen, true);
        check("watch released", model.watchId, "");
        check("unwatched ids", unwatched, ["w1"]);

        watchResponse = {
            "error": "no such directory",
            "code": "ENOENT"
        };
        model.path = "/tmp/missing";
        check("error code", model.errorCode, "ENOENT");
        check("error clears entries", names(), []);

        log.info(failures === 0 ? "FIXTURE PASS" : `FIXTURE FAIL (${failures})`);
        finish.running = true;
    }
}
