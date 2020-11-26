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

    CodeMirror.registerHelper("hint", "textplain", function(cm, options) {
        // Get cursor position
        let cur = cm.getCursor(0);

        // Get currentWord
        let currentWord = '';
        let worldFrom = 0;
        let text = cm.getLine(cur.line);
        if (cur.ch > 0) {
            let currentChar = '';
            let idx = cur.ch;
            do {
                worldFrom = idx;
                currentWord = currentChar + currentWord;
                idx--;
                if (idx < 0) {
                    break;
                }
                currentChar = text.substr(idx, 1)
            } while(currentChar !== ' ' && currentChar !== '(' )
        }
        let suggestions = options.completionList.filter(function (l) {
            return l.indexOf(currentWord) === 0;
        })
        return {
            list: suggestions,
            from: {
                line: cur.line,
                ch: worldFrom
            },
            to: CodeMirror.Pos(cur.line, worldFrom + currentWord.length)
        };
    });

    CodeMirror.registerHelper("hint", "cds", function (cm, options) {
        // Suggest list
        let cdsCompletionList = options.cdsCompletionList;

        // Get cursor position
        let cur = cm.getCursor(0);

        // Get current line
        let text = cm.getLine(cur.line);

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
        // Get current line
        let text = cm.getLine(cur.line);
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
        let text = cm.getLine(cur.line);

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
        let text = cm.getLine(cur.line);
        let autoCompleteResponse = {};

        let firstColon = text.indexOf(':');
        if (firstColon !== -1 && cur.ch > firstColon) {
            // autocomplete value
            autoCompleteResponse = autoCompleteValue(text, cur);
        } else if (firstColon === -1){
            // autocomplete key
            autoCompleteResponse = autoCompleteKey(text, options.schema, cur, cm);
        }

        return {
            list: autoCompleteResponse.sug,
            from: {
                line: cur.line,
                ch: autoCompleteResponse.fromCh
            },
            to: CodeMirror.Pos(cur.line, autoCompleteResponse.toCh)
        };

        function autoCompleteKey(text, schema, cur, cm) {
            // Find yaml level
            let depth = findDepth(text);
            if (depth === -1) {
                return {sug: []};
            }
            // Get suggestion
            return findKeySuggestion(depth, schema, cur, cm);
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

        function findKeySuggestion(depth, schema, cur, cm) {
            // Get all elements for this yaml level
            let eltMatchesLevel = schema.flatElements
                .filter(felt => felt.positions.findIndex(p => p.depth === depth) !== -1)
                .map(felt => {
                    felt.positions = felt.positions.filter(p => p.depth === depth);
                    return felt;
                });

            let suggestions = [];

            if (cur.line === 0) {
                suggestions = eltMatchesLevel.map(e => e.name + ': ');
            } else {
                let parents = findParent(cur.line -1, cm, depth);
                let lastParent = parents[parents.length-1];
                if (lastParent && parents[parents.length-1].indexOf('-') === 0) {
                    let lastParentTrimmed = lastParent.substr(1, lastParent.length).trimStart();
                    suggestions = schema.flatElements
                        .filter(elt => elt.positions.filter(p => p.depth === depth && p.parent[depth-1] === lastParentTrimmed).length > 0)
                        .map(elt => elt.name);
                } else {
                    // Find key to exclude from suggestion
                    let keyToExclude = findKeyToExclude(cur, cm, lastParent, schema);

                    // Filter suggestion ( match match and not in exclude array )
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
                        if (keepElt && keyToExclude.findIndex(e => e === elt.name) === -1) {
                            return elt.name + ': ';
                        }
                    }).filter(elt => elt);
                }
            }
            return {fromCh: depth*2, to: cur.line.length,sug: suggestions};
        }

        /**
         *  Find key to exclude from suggestion
         * @param cur Current position of the cursor in codemirror
         * @param cm Instance of codemirror file
         * @param lastParent Direct yaml parent
         * @param schema JSON schema
         * @returns {[]}  Array of string that contains all keys to exclude
         */
        function findKeyToExclude(cur, cm, lastParent, schema) {
            // Find neighbour to know which keys are already here
            let neighbour = findNeighbour(cur, cm);

            // Exclude key from oneOf.required
            let keyToExclude = [];
            let parent = schema.flatElements.find(e => e.name === lastParent);
            if (parent && parent.oneOf) {
                for (let i = 0; i < neighbour.length; i++) {
                    let key = neighbour[i];
                    if (!parent.oneOf[key]) {
                        continue
                    }
                    keyToExclude = Object.keys(parent.oneOf).filter( k => parent.oneOf[key].findIndex(kk => kk === k) === -1);
                }
            }

            // Add neighbour in exclude array
            keyToExclude.push(...neighbour);
            return keyToExclude;
        }

        /**
         * Find parent tree from cursor position
         * @param currentLine Current line position
         * @param cm Instance of the current file
         * @param depth Current depth reference
         * @returns {[]} Array of strings that contains the parent tree
         */
        function findParent(currentLine, cm, depth) {
            let parents = [];
            let refDepth = depth;
            for (let i = currentLine; i > 0; i--) {
                let currentText = cm.getLine(i);
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
            return parents;
        }

        /**
         * Find key at the same level with the same parent
         * @param cur Current cursor position
         * @param cm Instance of codemirror file
         * @returns {[]} Array of strings that contains all neighbour
         */
        function findNeighbour(cur, cm) {
            let neighbour = [];
            let givenLine = cm.getLine(cur.line);
            let nbOfSpaces = givenLine.length - givenLine.trimStart().length;

            if (cur.line > 0) {
                // find neighbour before
                for (let i = cur.line -1; i >= 0; i--) {
                    let currentLine = cm.getLine(i);
                    let currentText = currentLine.trimStart();
                    let currentSpace = currentLine.length - currentText.length;
                    if (currentSpace !== nbOfSpaces) {
                        // check if we are in a array
                        if (currentSpace + 2 !== nbOfSpaces || currentText.indexOf('-') !== 0) {
                            break;
                        }
                        currentText = currentText.substr(1, currentText.length).trimStart();
                    } else if (currentText.indexOf('-') === 0) {
                        continue;
                    }
                    neighbour.push(currentText.split(':')[0]);
                }
            }
            if (cur.line < cm.doc.size - 1) {
                // find neighbour before
                for (let i = cur.line + 1; i < cm.doc.size; i++) {
                    let currentLine = cm.getLine(i);
                    let currentSpace = currentLine.length - currentLine.trimStart();
                    if (currentSpace !== nbOfSpaces) {
                        break;
                    }
                    neighbour.push(currentLine.trimStart().split(':')[0]);
                }
            }
            return neighbour;
        }

        /**
         * Find yaml level
         * @param text Current text
         * @returns {number} level
         */
        function findDepth(text) {
            let spaceNumber = 0;

            for(let i=0; i<text.length; i++) {
                if (text[i] === ' ' || text[i] === '-') {
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
