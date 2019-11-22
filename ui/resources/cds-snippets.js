(function (mod) {
    if (typeof exports == "object" && typeof module == "object") // CommonJS
        mod(require("../../lib/codemirror"), require("../../mode/css/css"));
    else if (typeof define == "function" && define.amd) // AMD
        define(["../../lib/codemirror", "../../mode/css/css"], mod);
    else // Plain browser env
        mod(CodeMirror);
})(function (CodeMirror) {
    "use strict";

    CodeMirror.registerHelper("hint", "workflowAsCode", function (cm, options) {
        let suggest = [];
        let fromChar = 0;

        const pipPrefix = '    pipeline: ';
        const appPrefix = '    application: ';
        const envPrefix = '    environment: ';

        // Get cursor position
        let cur = cm.getCursor(0);

        if (!cur || !cm.doc.children[0].lines[cur.line]) {
            return null;
        }

        // Get current line
        let text = cm.doc.children[0].lines[cur.line].text;
        if (text.indexOf('@') === 0) {
            suggest = options.snippets;
        } else if (text.indexOf(pipPrefix) === 0) {
            suggest = options.suggests['pipelines'];
            fromChar = pipPrefix.length;
        } else if (text.indexOf(appPrefix) === 0) {
            suggest = options.suggests['applications'];
            fromChar = appPrefix.length;
        } else if (text.indexOf(envPrefix) === 0) {
            suggest = options.suggests['environments'];
            fromChar = envPrefix.length;
        }

        return {
            list: suggest,
            from: {
                line: cur.line,
                ch: fromChar
            },
            to: CodeMirror.Pos(cur.line, cur.ch)
        };
    });
});
