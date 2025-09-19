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
        let initialDepth = currentDepth;
        let parents = [];
        for (let i = position.lineNumber; i > 0; i--) {
            let currentText = model.getLineContent(i);
            if (currentText.indexOf(':') === -1) {
                continue
            }
            let currentLineDepth = Editor.findDepth(currentText);
            if (currentLineDepth < initialDepth) {
                    parents.push(currentText);
            }
        }
        parents.forEach((v, i) => {
            v = v.replace('- ', '  ');
            parents[i] = v
        })
        // Remove all parents after the one at level 0
        let indexRootParent = parents.findIndex(p => p.indexOf(' ') !== 0);
        if (indexRootParent === -1) {
            return [];
        }
        parents.splice(indexRootParent+1, parents.length-indexRootParent);

        let finalParent = [];
        let currenMaxSpaceLen = (position.column - 1);
        parents.forEach(v => {
            let spaceLen = v.length - v.trimStart().length;
            if (spaceLen < currenMaxSpaceLen) {
                currenMaxSpaceLen = spaceLen;
                let pkey = v.substring(0, v.indexOf(':')).trim();
                finalParent.unshift(pkey);
            }
            
        });
        return finalParent;
    }

    static findDepth(text) {
        let spaceNumber = 0;
        for (let i = 0; i < text.length; i++) {
            if (text[i] === ' ' || (text[i] === '-' && i == 0)) {
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
        if (depth === -1) {
            return 0
        }
        return depth;
    }

    static autoCompleteKey(model: ITextModel, position: IPosition, flatSchema: FlatSchema, currentDepthCursor: number) {
        let parents = Editor.findParent(model, position, currentDepthCursor);
        let finalDepth = parents.length;
       
        // Get suggestion
        return Editor.findKeySuggestion(model, position, parents, flatSchema, finalDepth);
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
        if (nbOfSpaces%2 !== 0) {
            return neighbour;
        }
        if (position.lineNumber > 0) {
            // find neighbour before
            let rootArrayElement = false;
      
            for (let i = position.lineNumber -1; i >= 1; i--) {
                let currentLine = model.getLineContent(i);
                let currentText = currentLine.trimStart();
                let currentSpace = currentLine.length - currentText.length;
                if (currentSpace < nbOfSpaces) {
                    if ( !(currentText.indexOf('-') === 0 && currentSpace + 2 === nbOfSpaces)) {
                        break;
                    }
                }
                if (currentSpace > nbOfSpaces) {
                    continue;
                }
                if (currentText.indexOf('-') === 0) {
                    currentText = currentText.substring(2, currentText.length);
                    rootArrayElement = true;
                }
                if (currentText.split(':')[0] === '' ) {
                    continue;
                }
                neighbour.push(currentText.split(':')[0]);
                if (rootArrayElement) {
                    break;
                }
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
        let suggestions = [];
        if (!depth) {
            depth = 0;
        }

       
        let lastParent = parents[parents.length - 1];
        // Find key to exclude from suggestion    
        let keyToExclude = Editor.findKeyToExclude(model, position, lastParent, schema);

        let eltMatchesLevel = schema.flatElements
        .filter(felt => felt.positions.findIndex(p => p.depth === depth) !== -1)
        .map(felt => {
            let i = JSON.parse(JSON.stringify(felt))
            i.positions = i.positions.filter(p => p.depth === depth);
            return i;
        });

        // Filter suggestion ( match match and not in exclude array )
        suggestions = eltMatchesLevel.map(elt => {
            let keepElt = false;
            for (let i = 0; i < elt.positions.length; i++) {
                let parentMatch = true;
                for (let j = 0; j < elt.positions[i].parent.length; j++) {
                    let regexp = elt.positions[i].parent[j];
                    if (regexp.indexOf('[') === -1) {
                        regexp = '^' + regexp + '$';
                    }
                    const regExp = RegExp(regexp);
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

        // Check if parent is an array and if we start a new element ( ie: in the suggestion there is all the children )
        if (depth > 0 && suggestions.length > 0) {
            let eltMatchesLevel = schema.flatElements
                .filter(felt => felt.name === lastParent && felt.positions.findIndex(p => p.depth === depth-1) !== -1);
            if (eltMatchesLevel.length === 1) {
                if (eltMatchesLevel[0].type.length === 1 && eltMatchesLevel[0].type[0] === 'array') {
                    let schemaType = eltMatchesLevel[0].schemaType;
                    if (schemaType && schema.flatTypes.has(schemaType) && schema.flatTypes.get(schemaType).length === suggestions.length) {
                       suggestions.forEach((v, i) => {
                        suggestions[i] = '- ' + v;
                       })
                    }
                }
            }
       } 
    
        return suggestions;
    }

}
