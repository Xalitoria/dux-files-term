pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    function list(path, includeHidden, callback) {
        DankFilesService.sendRequest("files.list", {
            "path": path,
            "includeHidden": includeHidden
        }, response => {
            if (response.error) {
                callback({
                    "error": response.error,
                    "code": response.code || ""
                });
                return;
            }
            const result = response.result || {};
            callback({
                "entries": result.entries || []
            });
        });
    }
}
