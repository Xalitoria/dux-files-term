pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.DankCommon.FileBrowser

ShellRoot {
    id: fixture

    readonly property var log: Log.scoped("fixture")
    property int failures: 0
    property var restoredMarkers: []

    function check(label, actual, expected) {
        const ok = JSON.stringify(actual) === JSON.stringify(expected);
        if (!ok) {
            failures++;
            log.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
            return;
        }
        log.info(`ok ${label}`);
    }

    readonly property NavigationHistory history: NavigationHistory {
        onRestored: marker => fixture.restoredMarkers = fixture.restoredMarkers.concat([marker])
    }

    Component.onCompleted: {
        history.restore({
            "path": "/home/me"
        });
        check("restore sets the path", history.path, "/home/me");
        check("restore clears the stacks", [history.canGoBack, history.canGoForward], [false, false]);

        history.go("/home/me/docs");
        check("go moves forward", history.path, "/home/me/docs");
        check("go remembers where it came from", history.serialize().back, ["/home/me"]);
        check("go can be undone", history.canGoBack, true);

        history.go("/home/me/docs");
        check("going to the current path is a no-op", history.serialize().back, ["/home/me"]);

        history.marker = {
            "contentY": 120
        };
        history.back();
        check("back returns to the previous path", history.path, "/home/me");
        check("back empties the back stack", history.canGoBack, false);
        check("back fills the forward stack", history.serialize().forward, ["/home/me/docs"]);

        history.forward();
        check("forward replays the path", history.path, "/home/me/docs");
        check("forward restores the scroll marker", restoredMarkers[restoredMarkers.length - 1], {
            "contentY": 120
        });

        history.go("/home/me/docs/reports");
        check("a new location clears the forward stack", history.canGoForward, false);

        history.up();
        check("up goes to the parent", history.path, "/home/me/docs");
        check("up is undoable", history.serialize().back, ["/home/me", "/home/me/docs", "/home/me/docs/reports"]);

        history.restore({
            "path": "/",
            "back": [],
            "forward": []
        });
        check("root has no parent", history.canGoUp, false);
        history.up();
        check("up at the root does nothing", history.path, "/");

        history.restore({
            "path": "/a/b",
            "back": ["/a"],
            "forward": ["/a/b/c"]
        });
        check("restore rebuilds both stacks", [history.canGoBack, history.canGoForward], [true, true]);
        check("a restored round trip serializes back", history.serialize(), {
            "path": "/a/b",
            "back": ["/a"],
            "forward": ["/a/b/c"]
        });

        if (failures === 0)
            log.info("FIXTURE PASS");
        Qt.callLater(Qt.quit);
    }
}
