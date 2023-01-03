import { Schema } from 'jsonschema';

export class JSONSchema implements Schema {

    static defPrefix = '#/definitions/';
    static refPrefix = '#/$defs/';

    static flat(schema: Schema): FlatSchema {
        let root = schema.$ref.replace(JSONSchema.defPrefix, '');
        let flatElts = new Array<FlatElement>();
        let flatTypes = new Map<string, Array<FlatTypeElement>>();

        JSONSchema.browse(schema, flatElts, flatTypes, root, []);
        let flatSchema = new FlatSchema();
        flatSchema.schema = schema;
        flatSchema.flatElements = flatElts;
        flatSchema.flatTypes = flatTypes;

        return flatSchema;
    }

    static browse(schema: Schema, flatSchema: Array<FlatElement>, flatTypes: Map<string, Array<FlatTypeElement>>, elt: string, tree: Array<string>): Schema[] {
        let currentType = elt.replace(JSONSchema.refPrefix, '');
        let defs = schema['$defs']
        let defElt = defs[currentType];
        let properties = defElt.properties;
        let oneOf = defElt.oneOf;
        if (!flatTypes.has(currentType)) {
            flatTypes.set(currentType, new Array<FlatTypeElement>());
        }
        if (properties) {
            Object.keys(properties).forEach( k => {
                if (properties[k].type && properties[k].type === 'object' && properties[k].patternProperties) {
                    let pp = properties[k].patternProperties;
                    if (pp['.*'] && pp['.*'].$ref) {
                        let newElt = pp['.*'].$ref.replace(JSONSchema.defPrefix, '');
                        JSONSchema.browse(schema, flatSchema, flatTypes, newElt, [...tree, k, '.*']);
                    }
                } else if (properties[k].type) {
                    let currentOneOf = new Array<Schema>();
                    if (properties[k].items && properties[k].items['$ref']) {
                        let newElt = properties[k].items['$ref'].replace(JSONSchema.defPrefix, '');
                        currentOneOf = JSONSchema.browse(schema, flatSchema, flatTypes, newElt, [...tree, k]);
                    }
                    JSONSchema.addElement(k, flatSchema, flatTypes.get(currentType), tree, [<string>properties[k].type], currentOneOf, properties[k]);
                } else if (properties[k].$ref) {
                    let newElt = properties[k].$ref.replace(JSONSchema.defPrefix, '');
                    let currentOneOf = JSONSchema.browse(schema, flatSchema, flatTypes, newElt, [...tree, k]);
                    JSONSchema.addElement(k, flatSchema, flatTypes.get(currentType), tree, ['object'], currentOneOf, properties[k]);
                } else {
                    let types = new Array<any>();
                    if (properties[k].oneOf) {
                        types = properties[k].oneOf.map(o => o.type).filter( o => o);
                    }
                    if (defElt.allOf) {
                        defElt.allOf.forEach(ao => {
                            if (ao?.then?.properties?.spec?.$ref) {
                                let newElt = ao?.then?.properties?.spec?.$ref.replace(JSONSchema.refPrefix, '');
                                JSONSchema.browse(schema, flatSchema, flatTypes, newElt, [...tree, k]);
                            }
                        });
                    }
                    JSONSchema.addElement(k, flatSchema, flatTypes.get(currentType), tree, types, null, properties[k], defElt.allOf);
                }
            });
        } else if (defElt.$ref) {
            let subRootElt = defElt.$ref.replace(JSONSchema.refPrefix, '');
            JSONSchema.browse(defElt, flatSchema, flatTypes, subRootElt, [...tree]);
        }
        return oneOf;
    }

    static addElement(k: string, flatSchema: Array<FlatElement>, typeItems: Array<FlatTypeElement>, tree: Array<string>, type: Array<string>, oneOf: Array<Schema>, properties, condition?: any) {
        if (type.length === 0) {
            type = ['object'];
        }
        if (type.length > 0) {
            let itemType = new FlatTypeElement();
            if (!type[0]) {
                type[0] = 'object';
            }
            itemType.type = type;
            if (itemType.type?.length === 1 && itemType.type[0] === 'object' && properties['$ref']) {
                itemType.type.push(properties['$ref'].replace(JSONSchema.refPrefix, ''));
            }
            itemType.name = k;
            itemType.enum = properties?.enum;
            itemType.formOrder = properties?.order;
            itemType.disabled = properties?.disabled;


            if (condition) {
                itemType.condition = new Array<FlatElementTypeCondition>();
                condition.forEach(ao => {
                    let c = new FlatElementTypeCondition();
                    if (ao?.if.properties) {
                        let keys = Object.keys(ao?.if.properties)
                        if (keys.length === 1) {
                            c.refProperty = keys[0];
                            c.conditionValue = ao.if.properties[keys[0]].const;
                        }
                    }
                    if (ao?.then?.properties?.spec?.$ref) {
                        let newElt = ao?.then?.properties?.spec?.$ref.replace(JSONSchema.refPrefix, '');
                        c.type = newElt;
                    }
                    itemType.condition.push(c);
                });
            }
            typeItems.push(itemType);
        }
        let flatElement = flatSchema.find(f => f.name === k);
        if (!flatElement) {
            flatElement = new FlatElement();
            flatElement.name = k;
            if (!type || type.length === 0) {
                type = ['object'];
            }
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
        if (condition) {
            flatElementPosition.condition = new Array<FlatElementTypeCondition>();
            condition.forEach(ao => {
                let c = new FlatElementTypeCondition();
                if (ao?.if.properties) {
                    let keys = Object.keys(ao?.if.properties)
                    if (keys.length === 1) {
                        c.refProperty = keys[0];
                        c.conditionValue = ao.if.properties[keys[0]].const;
                    }
                }
                if (ao?.then?.properties?.spec?.$ref) {
                    let newElt = ao?.then?.properties?.spec?.$ref.replace(JSONSchema.refPrefix, '');
                    c.type = newElt;
                }
                flatElementPosition.condition.push(c);
            });
        }
        if (properties) {
            if (properties.order) {
                flatElementPosition.formOrder = properties.order;
            }
            if (properties.enum) {
                flatElementPosition.enum = properties.enum;
            }
        }
        flatElement.positions.push(flatElementPosition);
    }
}

export class FlatSchema {
    schema: Schema;
    flatElements: Array<FlatElement>;
    flatTypes: Map<string, Array<FlatTypeElement>>;
}

export class FlatElement {
    name: string;
    positions: Array<FlatElementPosition>;
    type: Array<string>;
    children: Array<string>;
    oneOf: FlatElementsOneOfRequired;
}

export class FlatTypeElement {
    name: string;
    type: Array<string>;
    condition: Array<FlatElementTypeCondition>;
    formOrder: number;
    disabled: string;
    enum: string[];
}

export class FlatElementsOneOfRequired {
    name: {[key: string]: Array<string>};
}

export class FlatElementPosition {
    depth: number;
    parent: Array<string>;
    children: Array<string>;
    condition: Array<FlatElementTypeCondition>;
    formOrder: number;
    enum: string[];
}

export class FlatElementTypeCondition {
    refProperty: string
    conditionValue: string
    type: string
}


