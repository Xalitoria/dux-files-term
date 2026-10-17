pragma Singleton

import QtQuick
import Quickshell

Singleton {
    function tokens(query) {
        return (query ?? "").toLowerCase().split(/\s+/).filter(word => word !== "");
    }

    function score(query, title, subtitle, keywords) {
        const words = tokens(query);
        if (words.length === 0)
            return 0;
        const titleText = (title ?? "").toLowerCase();
        const titleWords = tokens(titleText);
        const subtitleText = (subtitle ?? "").toLowerCase();
        const keywordWords = tokens(keywords);
        let total = 0;
        for (const word of words) {
            let best = 0;
            switch (true) {
            case titleText.startsWith(word):
                best = 4;
                break;
            case titleWords.some(part => part.startsWith(word)):
                best = 3;
                break;
            case titleText.includes(word):
            case keywordWords.some(part => part.startsWith(word)):
                best = 2;
                break;
            case keywordWords.some(part => part.includes(word)):
            case subtitleText.includes(word):
                best = 1;
                break;
            }
            if (best === 0)
                return 0;
            total += best;
        }
        return total;
    }
}
