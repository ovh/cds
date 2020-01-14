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
        let autoCompleteResponse = {};

        let firstColon = text.indexOf(':');
        if (firstColon !== -1 && cur.ch > firstColon) {
            // autocomplete value
            autoCompleteResponse = autoCompleteValue(text, cur);
        } else if (firstColon === -1){
            // autocomplete key
            autoCompleteResponse = autoCompleteKey(text, options.schema, cur, cm.doc.children[0]);
        }

        return {
            list: autoCompleteResponse.sug,
            from: {
                line: cur.line,
                ch: autoCompleteResponse.fromCh
            },
            to: CodeMirror.Pos(cur.line, autoCompleteResponse.toCh)
        };

        function autoCompleteKey(text, schema, cur, fullText) {
            let depth = findDepth(text);
            if (depth === -1) {
                return {sug: []};
            }
            if (text.trimStart().indexOf('-') === 0) {
                return {sug: []};
            }
            return findKeySuggestion(depth, schema, cur, fullText);
        }

        function autoCompleteValue(text, cur) {
            const pipPrefix = 'pipeline: ';
            const appPrefix = 'application: ';
            const envPrefix = 'environment: ';
            let trimmedTrext = text.trimStart();

            let suggestions = [];
            if (trimmedTrext.indexOf(pipPrefix) === 0) {
                suggestions = options.suggests['pipelines'];
            } else if (trimmedTrext.indexOf(appPrefix) === 0) {
                suggestions = options.suggests['applications'];
            } else if (trimmedTrext.indexOf(envPrefix) === 0) {
                suggestions = options.suggests['environments'];
            }
            return {fromCh: text.indexOf(':') + 2, toCh: cur.ch, sug: suggestions}
        }

        function findKeySuggestion(depth, schema, cur, fullText) {
            let eltMatchesLevel = schema.flatElements
                .filter(felt => felt.positions.findIndex(p => p.depth === depth) !== -1)
                .map(felt => {
                    felt.positions = felt.positions.filter(p => p.depth === depth);
                    return felt;
                });

            let suggestions = [];

            // Find parents
            if (cur.line === 0) {
                suggestions = eltMatchesLevel.map(e => e.name + ': ');
            } else {
                let currentLine = cur.line -1;
                let parents = [];
                let refDepth = depth;
                for (let i = currentLine; i > 0; i--) {
                    let currentText = fullText.lines[i].text;
                    if (currentText.indexOf(':') === -1) {
                        continue
                    }
                    // if has key, find indentation
                    let currentLintDepth = findDepth(currentText);
                    if (currentLintDepth >= refDepth) {
                        continue
                    }
                    // find parent key
                    let pkey = currentText.substr(0, currentText.indexOf(':')).trimStart();
                    parents.unshift(pkey);
                    refDepth = currentLintDepth;
                    if (refDepth === 0) {
                        break;
                    }
                }

                suggestions = eltMatchesLevel.map(elt => {
                    let keepElt = false;
                    for(let i = 0; i < elt.positions.length; i++) {
                        let parentMatch = true;
                        for(let j = 0; j < elt.positions[i].parent.length; j++) {
                            const regExp = RegExp(elt.positions[i].parent[j]);
                            if (!regExp.test(parents[j])) {
                                parentMatch = false;
                                break;
                            }
                        }
                        if (parentMatch) {
                            keepElt = true;
                            break;
                        }
                    }
                    if (keepElt) {
                        return elt.name + ': ';
                    }
                }).filter(elt => elt);

            }
            return {fromCh: depth*2, to: cur.line.length,sug: suggestions};
        }

        function findDepth(text) {
            let spaceNumber = 0;
            for(let i=0; i<text.length; i++) {
                if (text[i] === ' ') {
                    spaceNumber++;
                    continue;
                } else {
                    break;
                }
            }
            let depth = -1;
            if (spaceNumber%2 === 0) {
                depth = spaceNumber/2;
            }
            return depth;
        }
    });
});
