/**
 * Hint addon for codemirror
 */

(function(mod) {
    if (typeof exports == "object" && typeof module == "object") // CommonJS
        mod(require("../../lib/codemirror"), require("../../mode/css/css"));
    else if (typeof define == "function" && define.amd) // AMD
        define(["../../lib/codemirror", "../../mode/css/css"], mod);
    else // Plain browser env
        mod(CodeMirror);
})(function(CodeMirror) {
    "use strict";

    CodeMirror.registerHelper("hint", "cds", function(cm, options) {
        // Suggest list
        var cdsCompletionList = options.cdsCompletionList;

        // Get cursor position
        var cur = cm.getCursor(0);

        // Get current line
        var text = cm.doc.children[0].lines[cur.line].text;

        // Show nothing if there is no  {{. on the line
        if (text.indexOf('{{.') === -1) {
            return null;
        }

        var areaBefore = text.substring(0, cur.ch);
        if (areaBefore.lastIndexOf('{{.') < areaBefore.lastIndexOf('}}')) {
            return null
        }

        // If the previous value was already completed
        if (text.lastIndexOf('}}') !== -1 && text.lastIndexOf('}}') >= cur.ch) {
            cdsCompletionList = cdsCompletionList.map(function (suggest) {
                return suggest.replace('}}', '');
            });
        }

        return {
            list: cdsCompletionList.filter(function (l) {
                return l.indexOf(areaBefore.substring(areaBefore.lastIndexOf('{{.'))) !== -1;
            }),
            from: { line: cur.line, ch: areaBefore.lastIndexOf('{{.')},
            to: CodeMirror.Pos(cur.line, cur.ch)
        };
    });

    CodeMirror.registerHelper("hint", "condition", function(cm, options) {
        // Suggest list
        var cdsCompletionList = options.cdsCompletionList;

        // Get cursor position
        var cur = cm.getCursor(0);

        // Get current line
        var text = cm.doc.children[0].lines[cur.line].text;
        // Show nothing if there is no  {{. on the line
        if (text.indexOf('cds_') === -1) {
            return null;
        }

        var areaBefore = text.substring(0, cur.ch);

        return {
            list: cdsCompletionList.filter(function (l) {
                return l.indexOf(areaBefore.substring(areaBefore.lastIndexOf('cds_'))) !== -1;
            }),
            from: { line: cur.line, ch: areaBefore.lastIndexOf('cds_')},
            to: CodeMirror.Pos(cur.line, cur.ch)
        };
    });

    CodeMirror.registerHelper("hint", "payload", function(cm, options) {
        var branchPrefix = '"git.branch":';
        // Suggest list
        var payloadCompletionList = options.payloadCompletionList;

        // Get cursor position
        var cur = cm.getCursor(0);
        var from = 0;

        // Get current line
        var text = cm.doc.children[0].lines[cur.line].text;

        // Show nothing if there is no branchPrefix on the line
        if (text.indexOf(branchPrefix) === -1) {
            return null;
        }

        var lastIndexOfBranchPrefix = text.lastIndexOf(branchPrefix);
        var areaBefore = text.substring(0, cur.ch);
        var areaAfterPrefix = text.substring(lastIndexOfBranchPrefix + branchPrefix.length + 1);
        var lastIndexOfComma = areaAfterPrefix.indexOf(',');

        if (lastIndexOfComma === -1) {
            lastIndexOfComma += text.length + 1;
        } else if (lastIndexOfComma === 0) {
            lastIndexOfComma += (lastIndexOfBranchPrefix + branchPrefix.length);
        } else {
            lastIndexOfComma += (lastIndexOfBranchPrefix + branchPrefix.length + 1);
        }

        return {
            list: payloadCompletionList,
            from: { line: cur.line, ch: lastIndexOfBranchPrefix + branchPrefix.length + 1},
            to: CodeMirror.Pos(cur.line, lastIndexOfComma)
        };
    });
});
