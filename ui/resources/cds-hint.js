/**
 * Hint addon for codemirror
 */

(function (mod) {
    if (typeof exports == "object" && typeof module == "object") // CommonJS
        mod(require("../../lib/codemirror"), require("../../mode/css/css"));
    else if (typeof define == "function" && define.amd) // AMD
        define(["../../lib/codemirror", "../../mode/css/css"], mod);
    else // Plain browser env
        mod(CodeMirror);
})(function (CodeMirror) {
    "use strict";

    CodeMirror.registerHelper("hint", "cds", function (cm, options) {
        // Suggest list
        let cdsCompletionList = options.cdsCompletionList;

        // Get cursor position
        let cur = cm.getCursor(0);

        // Get current line
        let line = cm.doc.children[0].lines[cur.line];
        let text = '';

        if (!line) {
            return null;
        }
        text = line.text;

        // Show nothing if there is no  {{. on the line
        if (text.indexOf('{{.') === -1) {
            return null;
        }

        let areaBefore = text.substring(0, cur.ch);
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
            from: {
                line: cur.line,
                ch: areaBefore.lastIndexOf('{{.')
            },
            to: CodeMirror.Pos(cur.line, cur.ch)
        };
    });

    CodeMirror.registerHelper("hint", "condition", function (cm, options) {
        let cdsPrefix = 'cds_';
        let workflowPrefix = 'workflow_';
        let gitPrefix = 'git_';
        // Suggest list
        let cdsCompletionList = options.cdsCompletionList;

        // Get cursor position
        let cur = cm.getCursor(0);

        if (!cur || !cm.doc.children[0].lines[cur.line]) {
            return null;
        }

        // Get current line
        let text = cm.doc.children[0].lines[cur.line].text;
        if (text.indexOf(cdsPrefix) === -1 && text.indexOf(workflowPrefix) === -1 && text.indexOf(gitPrefix) === -1) {
            return null;
        }

        let areaBefore = text.substring(0, cur.ch);
        let cdsPrefixCh = areaBefore.lastIndexOf(cdsPrefix);
        let workflowPrefixCh = areaBefore.lastIndexOf(workflowPrefix);
        let gitPrefixCh = areaBefore.lastIndexOf(gitPrefix);

        let ch = Math.max(cdsPrefixCh, workflowPrefixCh, gitPrefixCh);
        return {
            list: cdsCompletionList.filter(function (l) {
                return l.indexOf(areaBefore.substring(areaBefore.lastIndexOf(cdsPrefix))) !== -1 ||
                    l.indexOf(areaBefore.substring(areaBefore.lastIndexOf(workflowPrefix))) !== -1 ||
                    l.indexOf(areaBefore.substring(areaBefore.lastIndexOf(gitPrefix))) !== -1;
            }),
            from: {
                line: cur.line,
                ch: ch
            },
            to: CodeMirror.Pos(cur.line, cur.ch)
        };
    });

    CodeMirror.registerHelper("hint", "payload", function (cm, options) {
        let branchPrefix = '"git.branch":';
        let tagPrefix = '"git.tag":';
        let repoPrefix = '"git.repository":';
        // Suggest list
        let payloadCompletionList = [];

        // Get cursor position
        let cur = cm.getCursor(0);

        // Get current line
        if (!cur || !cm.doc.children[0].lines[cur.line]) {
            return null;
        }

        let text = cm.doc.children[0].lines[cur.line].text;
        let prefix = "";

        if (!options.payloadCompletionList) {
            return null;
        }

        switch (true) {
            case text.indexOf(branchPrefix) !== -1:
                payloadCompletionList = options.payloadCompletionList.branches;
                prefix = branchPrefix;
                break;
            case text.indexOf(repoPrefix) !== -1:
                payloadCompletionList = options.payloadCompletionList.repositories;
                prefix = repoPrefix;
                break;
            case text.indexOf(tagPrefix) !== -1:
                payloadCompletionList = options.payloadCompletionList.tags;
                prefix = tagPrefix;
                break;
            default:
                return null;
        }

        let lastIndexOfPrefix = text.lastIndexOf(prefix);
        let areaAfterPrefix = text.substring(lastIndexOfPrefix + prefix.length + 1);
        let lastIndexOfComma = areaAfterPrefix.indexOf(',');
        let indexOfComma = text.indexOf(',');
        if (indexOfComma !== -1 && cur.ch >= indexOfComma) {
            return null;
        }

        if (lastIndexOfComma === -1) {
            lastIndexOfComma += text.length + 1;
        } else if (lastIndexOfComma === 0) {
            lastIndexOfComma += (lastIndexOfPrefix + prefix.length);
        } else {
            lastIndexOfComma += (lastIndexOfPrefix + prefix.length + 1);
        }
        let inc = 0;

        if (text.indexOf(prefix + ' ') !== -1) {
            inc = 1;
        }

        return {
            list: payloadCompletionList,
            from: {
                line: cur.line,
                ch: lastIndexOfPrefix + prefix.length + inc
            },
            to: CodeMirror.Pos(cur.line, lastIndexOfComma)
        };
    });
    CodeMirror.registerHelper("hint", "asCode", function (cm, options) {
        
        // Get cursor position
        let cur = cm.getCursor(0);

        // Get current line
        if (!cur || !cm.doc.children[0].lines[cur.line]) {
            return null;
        }
        let text = cm.doc.children[0].lines[cur.line].text;
        let autoCompleteList = [];
        let ch = 0;

        let textToComplete = text.substring(0, cur.ch);

        // Detect : and quote

        let indentLevel = 0;
        let parentTree = [];



        return {
            list: autoCompleteList,
            from: {
                line: cur.line,
                ch: ch
            },
            to: CodeMirror.Pos(cur.line, ch)
        };
    });
});
