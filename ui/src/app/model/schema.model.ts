import { Schema } from 'jsonschema';

export class JSONSchema implements Schema {

    static defPrefix = '#/definitions/';
    static refPrefix = '#/$defs/';
    static flat(schema: Schema): FlatSchema {
        let root = schema.$ref.replace(JSONSchema.defPrefix, '');
        let flatElts = new Array<FlatElement>();
        JSONSchema.browse(schema, flatElts, root, []);

        let flatSchema = new FlatSchema();
        flatSchema.schema = schema;
        flatSchema.flatElements = flatElts;
        return flatSchema;
    }

    static browse(schema: Schema, flatSchema: Array<FlatElement>, elt: string, tree: Array<string>, condition?: any): Schema[] {
        let currentType = elt.replace(JSONSchema.refPrefix, '');
        let defs = schema['$defs']
        let defElt = defs[currentType];
        let properties = defElt.properties;
        let oneOf = defElt.oneOf;
        if (properties) {
            Object.keys(properties).forEach( k => {
                if (properties[k].type && properties[k].type === 'object' && properties[k].patternProperties) {
                    let pp = properties[k].patternProperties;
                    if (pp['.*'] && pp['.*'].$ref) {
                        let newElt = pp['.*'].$ref.replace(JSONSchema.defPrefix, '');
                        JSONSchema.browse(schema, flatSchema, newElt, [...tree, k, '.*']);
                    }
                } else if (properties[k].type) {
                    let currentOneOf = new Array<Schema>();
                    if (properties[k].items && properties[k].items['$ref']) {
                        let newElt = properties[k].items['$ref'].replace(JSONSchema.defPrefix, '');
                        currentOneOf = JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                    }
                    JSONSchema.addElement(k, flatSchema, tree, [<string>properties[k].type], currentOneOf, condition);
                } else if (properties[k].$ref) {
                    let newElt = properties[k].$ref.replace(JSONSchema.defPrefix, '');
                    let currentOneOf = JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                    JSONSchema.addElement(k, flatSchema, tree, ['object'], currentOneOf, condition);
                } else {
                    let types = new Array<any>();
                    if (properties[k].oneOf) {
                        types = properties[k].oneOf.map(o => o.type).filter( o => o);
                    }
                    if (defElt.allOf) {
                        defElt.allOf.forEach(ao => {
                            if (ao?.then?.properties?.spec?.$ref) {
                                let newElt = ao?.then?.properties?.spec?.$ref.replace(JSONSchema.refPrefix, '');
                                JSONSchema.browse(schema, flatSchema, newElt, [...tree, k], ao?.if?.properties);
                            }
                        });
                    }
                    JSONSchema.addElement(k, flatSchema, tree, types, null, condition);
                }
            });
        } else if (defElt.$ref) {
            let subRootElt = defElt.$ref.replace(JSONSchema.refPrefix, '');
            JSONSchema.browse(defElt, flatSchema, subRootElt, [...tree], condition);
        }
        return oneOf;
    }

    static addElement(k: string, flatSchema: Array<FlatElement>, tree: Array<string>, type: Array<string>, oneOf: Array<Schema>, condition?: any) {
        let flatElement = flatSchema.find(f => f.name === k);
        if (!flatElement) {
            flatElement = new FlatElement();
            flatElement.name = k;
            flatElement.type = type;
            flatElement.positions = new Array<FlatElementPosition>();
            flatElement.oneOf = new FlatElementsOneOfRequired();
            if (oneOf) {
                oneOf.forEach(o => {
                    if (!o.required) {
                        return;
                    }
                    o.required.forEach(r => {
                        if (!flatElement.oneOf[r]) {
                            flatElement.oneOf[r] = [];
                        }
                        flatElement.oneOf[r].push(...o.required);
                    });
                });
            }
            flatSchema.push(flatElement);

        }
        let flatElementPosition = new FlatElementPosition();
        flatElementPosition.depth = tree.length;
        flatElementPosition.parent = tree;
        flatElementPosition.condition = condition;
        flatElement.positions.push(flatElementPosition);
    }
}

export class FlatSchema {
    schema: Schema;
    flatElements: Array<FlatElement>;
}

export class FlatElement {
    name: string;
    positions: Array<FlatElementPosition>;
    type: Array<string>;
    children: Array<string>;
    oneOf: FlatElementsOneOfRequired;
}

export class FlatElementsOneOfRequired {
    name: {[key: string]: Array<string>};
}

export class FlatElementPosition {
    depth: number;
    parent: Array<string>;
    children: Array<string>;
    condition: any;
}


