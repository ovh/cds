import {Schema} from 'jsonschema';

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
        flatSchema.flatTypes = JSONSchema.getTypeMap(schema);
        return flatSchema;
    }

    static browse(schema: Schema, flatSchema: Array<FlatElement>, elt: string, tree: Array<string>): Schema[] {
        let currentType = elt.replace(JSONSchema.refPrefix, '');
        let defs = schema['$defs']
        let defElt = defs[currentType];
        let properties = defElt.properties;
        let oneOf = defElt.oneOf;
        if (properties) {
            Object.keys(properties).forEach(k => {
                if (properties[k].type && properties[k].type === 'object' && properties[k].patternProperties) {
                    let pp = properties[k].patternProperties;
                    let ppKeys = Object.keys(pp)
                    if (ppKeys.length === 1 && pp[ppKeys[0]].$ref) {
                        let newElt = pp[ppKeys[0]].$ref.replace(JSONSchema.defPrefix, '');
                        JSONSchema.browse(schema, flatSchema, newElt, [...tree, k, ppKeys[0]]);
                    }
                } else if (properties[k].type) {
                    let currentOneOf = new Array<Schema>();
                    if (properties[k].items && properties[k].items['$ref']) {
                        let newElt = properties[k].items['$ref'].replace(JSONSchema.defPrefix, '');
                        currentOneOf = JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                    }
                    JSONSchema.addElement(k, flatSchema, tree, [<string>properties[k].type], currentOneOf, properties[k]);
                } else if (properties[k].$ref) {
                    let newElt = properties[k].$ref.replace(JSONSchema.defPrefix, '');
                    let currentOneOf = JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                    JSONSchema.addElement(k, flatSchema, tree, ['object'], currentOneOf, properties[k]);
                } else {
                    let types = new Array<any>();
                    if (properties[k].oneOf) {
                        types = properties[k].oneOf.map(o => o.type).filter(o => o);
                    }
                    if (defElt.allOf) {
                        defElt.allOf.forEach(ao => {
                            if (ao?.then?.properties?.spec?.$ref) {
                                let newElt = ao?.then?.properties?.spec?.$ref.replace(JSONSchema.refPrefix, '');
                                JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                            }
                        });
                    }
                    JSONSchema.addElement(k, flatSchema, tree, types, null, properties[k], defElt.allOf);
                }
            });
        } else if (defElt.$ref) {
            let subRootElt = defElt.$ref.replace(JSONSchema.refPrefix, '');
            JSONSchema.browse(defElt, flatSchema, subRootElt, [...tree]);
        }
        return oneOf;
    }

    static getTypeMap(schema: Schema): Map<string, Array<FlatTypeElement>> {
        let root = schema.$ref.replace(JSONSchema.defPrefix, '');
        let flatTypes = new Map<string, Array<FlatTypeElement>>();
        JSONSchema.flattenTypes(schema, root, flatTypes)
        return flatTypes
    }

    static flattenTypes(schema: Schema, elt: string, flatTypes: Map<string, Array<FlatTypeElement>>) {
        let currentType = elt.replace(JSONSchema.refPrefix, '').replace(JSONSchema.defPrefix, '');
        let defs = schema['$defs']
        let defElt = defs[currentType];
        let properties = defElt.properties;
        if (!flatTypes.has(currentType)) {
            flatTypes.set(currentType, new Array<FlatTypeElement>());
        }
        if (properties) {
            Object.keys(properties).forEach(k => {
                // MAP
                if (properties[k].type && properties[k].type === 'object' && properties[k].patternProperties) {
                    let pp = properties[k].patternProperties;
                    let mapKeys = Object.keys(pp)
                    if (mapKeys.length === 1 && pp[mapKeys[0]].$ref) {
                        let newElt = pp[mapKeys[0]].$ref.replace(JSONSchema.defPrefix, '');
                        JSONSchema.flattenTypes(schema, newElt, flatTypes);
                        flatTypes.get(currentType).push(JSONSchema.toFlatTypeElement(k, ['map', 'string', mapKeys[0], newElt.replace(JSONSchema.refPrefix, '')], properties[k]))
                    } else if (mapKeys.length === 1 && pp[mapKeys[0]].type) {
                        let newEltType = pp[mapKeys[0]].type;
                        flatTypes.get(currentType).push(JSONSchema.toFlatTypeElement(k, ['map', 'string', mapKeys[0], newEltType], properties[k]))
                    }
                }
                // Simple TYPE (string, number, array)
                else if (properties[k].type) {
                    flatTypes.get(currentType).push(JSONSchema.toFlatTypeElement(k, [<string>properties[k].type], properties[k]))
                    if (properties[k].type == 'array' && properties[k].items.$ref) {
                        JSONSchema.flattenTypes(schema, properties[k].items.$ref, flatTypes);
                    }
                }
                // Refs type
                else if (properties[k].$ref) {
                    flatTypes.get(currentType).push(JSONSchema.toFlatTypeElement(k, ['object'], properties[k]));
                    JSONSchema.flattenTypes(schema, properties[k].$ref, flatTypes);
                }
                // No type, check oneOf and allOf
                else {
                    let types = new Array<any>();
                    if (properties[k].oneOf) {
                        types = properties[k].oneOf.map(o => o.type).filter(o => o);
                    }
                    if (defElt.allOf) {
                        defElt.allOf.forEach(ao => {
                            if (ao?.then?.properties?.spec?.$ref) {
                                JSONSchema.flattenTypes(schema, ao?.then?.properties?.spec?.$ref, flatTypes);
                            }
                        });
                        types = ['object'];
                    }
                    flatTypes.get(currentType).push(JSONSchema.toFlatTypeElement(k, types, properties[k], defElt.allOf));
                }
            });
        } else if (defElt.$ref) {
            let subRootElt = defElt.$ref.replace(JSONSchema.refPrefix, '');
            JSONSchema.flattenTypes(defElt, subRootElt, flatTypes);
        }
    }

    static toFlatTypeElement(name: string, type: Array<string>, properties, condition?: any) {
        let itemType = new FlatTypeElement();
        itemType.type = type;
        if (itemType.type?.length === 1 && itemType.type[0] === 'object' && properties['$ref']) {
            itemType.type.push(properties['$ref'].replace(JSONSchema.refPrefix, ''));
        }
        if (type[0] === 'array') {
            if (properties['items'].$ref) {
                itemType.type.push(properties['items'].$ref.replace(JSONSchema.refPrefix, ''));
            } else {
                itemType.type.push(properties['items'].type);
            }
        }
        itemType.name = name;
        itemType.enum = properties?.enum;
        itemType.formOrder = properties?.order;
        itemType.code = properties?.code;
        itemType.disabled = properties?.disabled;
        itemType.description = properties?.description;
        itemType.pattern = properties?.pattern;
        itemType.onchange = properties?.onchange;
        itemType.mode = properties?.mode;
        itemType.prefix = properties?.prefix;

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
        return itemType

    }

    static addElement(k: string, flatSchema: Array<FlatElement>, tree: Array<string>, type: Array<string>, oneOf: Array<Schema>, properties, condition?: any) {
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
            if (properties.code) {
                flatElementPosition.code = properties.code;
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
    description: string;
    formOrder: number;
    disabled: string;
    enum: string[];
    pattern: string;
    onchange: string;
    mode: string;
    prefix: string;
    code: boolean;
}

export class FlatElementsOneOfRequired {
    name: { [key: string]: Array<string> };
}

export class FlatElementPosition {
    depth: number;
    parent: Array<string>;
    children: Array<string>;
    condition: Array<FlatElementTypeCondition>;
    formOrder: number;
    enum: string[];
    code: boolean;
}

export class FlatElementTypeCondition {
    refProperty: string
    conditionValue: string
    type: string
}


