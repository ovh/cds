import {editor, IPosition, Range} from "monaco-editor";
import {FlatElement, FlatElementPosition, FlatSchema} from "./schema.model";
import ITextModel = editor.ITextModel;

export class Editor {

    static completionProvider(monaco): any {
        return {
            triggerCharacters: [' ', ':', '\n'],
            async provideCompletionItems(model, position) {
                let flatSchema: FlatSchema = monaco.languages.json.jsonDefaults._diagnosticsOptions.schemas[0].schema;
                const wordInfo = model.getWordUntilPosition(position);
                let currentLine = model.getLineContent(position.lineNumber);
                let firstColon = currentLine.indexOf(':');
                let result = [];
                let currentDepthCursor = Editor.findDepth(model.getLineContent(position.lineNumber));
                if (firstColon !== -1 && (position.column - 1) > firstColon) {
                    // Value suggestion : manage enum
                    result = Editor.autoCompleteValue(model, position, flatSchema, currentDepthCursor);
                } else {
                    if (currentDepthCursor === -1) {
                        return {
                            incomplete: false,
                            suggestions: [],
                        };
                    }
                    result = Editor.autoCompleteKey(model, position, flatSchema, currentDepthCursor);
                }
                if (!result) {
                    return null;
                }
                return {
                    incomplete: false,
                    suggestions: result.map(r => {
                        return {
                            label: r,
                            kind: 13,//CompletionItemKind.Value;
                            insertText: r,
                            range: new Range(
                                position.lineNumber,
                                wordInfo.startColumn,
                                position.lineNumber,
                                wordInfo.endColumn,
                            )
                        }
                    }),
                };
            },
        }
    }

    static autoCompleteValue(model: ITextModel, position: IPosition, flatSchema: FlatSchema, currentDepth): string[] {
        let currentLineContent = model.getLineContent(position.lineNumber);
        let key = currentLineContent.replace(':', '').trim();
        let parents = Editor.findParent(model, position, currentDepth);
        let flatEltPosition = Editor.findElement(key, parents, flatSchema.flatElements);
        if (!flatEltPosition) {
            return [];
        }
        return flatEltPosition.enum;
    }

    static findElement(key: string, parents: Array<string>, flatElts: FlatElement[]): FlatElementPosition {
        let currentEltPosition = new FlatElementPosition();
        all: for (let i=0; i<flatElts.length; i++) {
            let flatElt = flatElts[i];
            if (flatElt.name !== key) {
                continue
            }
            posLoop: for (let j=0; j<flatElt.positions.length; j++) {
                let eltPos = flatElt.positions[j];
                if (parents.length !== eltPos.parent.length) {
                    continue;
                }
                for (let k = 0; k<eltPos.parent.length; k++) {
                    if (!parents[k].match(eltPos.parent[k])) {
                        continue posLoop
                    }
                }
                currentEltPosition = eltPos;
                break all;
            }
        }
        return currentEltPosition;
    }

    static findParent(model: ITextModel, position: IPosition, currentDepth: number) {
        let parents = [];
        for (let i = position.lineNumber; i > 0; i--) {
            let currentText = model.getLineContent(i);
            if (currentText.indexOf(':') === -1) {
                continue
            }
            // if has key, find indentation
            let currentLintDepth = Editor.findDepth(currentText);
            if (currentLintDepth >= currentDepth) {
                continue
            }
            // find parent key
            let pkey = currentText.substring(0, currentText.indexOf(':')).trim();
            parents.unshift(pkey);
            currentDepth = currentLintDepth;
            if (currentDepth === 0) {
                break;
            }
        }
        return parents;
    }

    static findDepth(text) {
        let spaceNumber = 0;
        for (let i = 0; i < text.length; i++) {
            if (text[i] === ' ' || text[i] === '-') {
                spaceNumber++;
                continue;
            } else {
                break;
            }
        }
        let depth = -1;
        if (spaceNumber % 2 === 0) {
            depth = spaceNumber / 2;
        }
        return depth;
    }

    static autoCompleteKey(model: ITextModel, position: IPosition, flatSchema: FlatSchema, currentDepthCursor: number) {
        let parents = Editor.findParent(model, position, currentDepthCursor);
        // Get suggestion
        return Editor.findKeySuggestion(model, position, parents, flatSchema, currentDepthCursor);
    }

    // Exclude key that are already here
    static findKeyToExclude(model: ITextModel, position: IPosition, lastParent: string, flatSchema: FlatSchema) {
        // Find neighbour to know which keys are already here
        let neighbour = Editor.findNeighbour(model, position);

        // Exclude key from oneOf.required
        let keyToExclude = [];
        let parent = flatSchema.flatElements.find(e => e.name === lastParent);
        if (parent && parent.oneOf) {
            for (let i = 0; i < neighbour.length; i++) {
                let key = neighbour[i];
                if (!parent.oneOf[key]) {
                    continue
                }
                keyToExclude = Object.keys(parent.oneOf).filter(k => parent.oneOf[key].findIndex(kk => kk === k) === -1);
            }
        }

        // Add neighbour in exclude array
        keyToExclude.push(...neighbour);
        return keyToExclude;
    }

    // Find all key at the same level/ same parent
    static findNeighbour(model: ITextModel, position: IPosition) {
        let neighbour = [];
        let givenLine = model.getLineContent(position.lineNumber);
        let nbOfSpaces = givenLine.length - givenLine.trim().length;
        if (position.lineNumber > 0) {
            // find neighbour before
            for (let i = position.lineNumber; i >= 1; i--) {
                let currentLine = model.getLineContent(i);
                let currentText = currentLine.trim();
                let currentSpace = currentLine.length - currentText.length;
                if (currentSpace !== nbOfSpaces) {
                    // check if we are in a array
                    if (currentSpace + 2 !== nbOfSpaces || currentText.indexOf('-') !== 0) {
                        break;
                    }
                    currentText = currentText.substr(1, currentText.length).trim();
                } else if (currentText.indexOf('-') === 0) {
                    continue;
                }
                neighbour.push(currentText.split(':')[0]);
            }
        }
        if (position.lineNumber <= model.getLineCount()) {
            if (nbOfSpaces !== 0) {
                for (let i = position.lineNumber; i <= model.getLineCount(); i++) {
                    let currentLine = model.getLineContent(i);
                    let currentSpace = currentLine.length - currentLine.trim().length;
                    if (currentSpace !== nbOfSpaces) {
                        break;
                    }
                    neighbour.push(currentLine.trim().split(':')[0]);
                }
            } else {
                for (let i = 1; i <= model.getLineCount(); i++) {
                    let currentLine = model.getLineContent(i);
                    let currentSpace = currentLine.length - currentLine.trim().length;
                    if (currentSpace == nbOfSpaces) {
                        neighbour.push(currentLine.trim().split(':')[0]);
                    }
                }
            }
        }
        return neighbour;
    }

    static findKeySuggestion(model: ITextModel, position: IPosition, parents: string[], schema: FlatSchema, depth: number) {
        let eltMatchesLevel = schema.flatElements
            .filter(felt => felt.positions.findIndex(p => p.depth === depth) !== -1)
            .map(felt => {
                felt.positions = felt.positions.filter(p => p.depth === depth);
                return felt;
            });
        let suggestions = [];

        let lastParent = parents[parents.length - 1];
        if (lastParent && parents[parents.length - 1].indexOf('-') === 0) {
            let lastParentTrimmed = lastParent.substring(1, lastParent.length).trim();
            suggestions = schema.flatElements
                .filter(elt => elt.positions.filter(p => p.depth === depth && p.parent[depth - 1] === lastParentTrimmed).length > 0)
                .map(elt => elt.name);
        } else {
            if (lastParent && parents[parents.length - 1].indexOf('-') === 0) {
                let lastParentTrimmed = lastParent.substring(1, lastParent.length).trim();
                suggestions = schema.flatElements
                    .filter(elt => elt.positions.filter(p => p.depth === depth && p.parent[depth - 1] === lastParentTrimmed).length > 0)
                    .map(elt => elt.name);
            } else {
                // Find key to exclude from suggestion
                let keyToExclude = Editor.findKeyToExclude(model, position, lastParent, schema);

                // Filter suggestion ( match match and not in exclude array )
                suggestions = eltMatchesLevel.map(elt => {
                    let keepElt = false;
                    for (let i = 0; i < elt.positions.length; i++) {
                        let parentMatch = true;
                        for (let j = 0; j < elt.positions[i].parent.length; j++) {
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

        return suggestions;
    }

}
