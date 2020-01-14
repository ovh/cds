import { Schema } from 'jsonschema';

export class JSONSchema implements Schema {

    static defPrefix = '#/definitions/';
    static flat(schema: Schema): FlatSchema {
        let root = schema.$ref.replace(JSONSchema.defPrefix, '');
        let flatElts = new Array<FlatElement>();
        JSONSchema.browse(schema, flatElts, root, []);

        let flatSchema = new FlatSchema();
        flatSchema.schema = schema;
        flatSchema.flatElements = flatElts;
        return flatSchema;
    }

    static browse(schema: Schema, flatSchema: Array<FlatElement>, elt: string, tree: Array<string>) {
        let properties = schema.definitions[elt].properties;
        if (properties) {
            Object.keys(properties).forEach( k => {
                if (properties[k].type && properties[k].type === 'object' && properties[k].patternProperties) {
                    let pp = properties[k].patternProperties;
                    if (pp['.*'] && pp['.*'].$ref) {
                        let newElt = pp['.*'].$ref.replace(JSONSchema.defPrefix, '');
                        JSONSchema.browse(schema, flatSchema, newElt, [...tree, k, '.*'])
                    }
                }
                if (properties[k].type) {
                    JSONSchema.addElement(k, flatSchema, tree);
                    if (properties[k].items && properties[k].items['$ref']) {
                        let newElt = properties[k].items['$ref'].replace(JSONSchema.defPrefix, '');
                        JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                    }
                }
                if (properties[k].$ref) {
                    JSONSchema.addElement(k, flatSchema, tree);
                    let newElt = properties[k].$ref.replace(JSONSchema.defPrefix, '');
                    JSONSchema.browse(schema, flatSchema, newElt, [...tree, k]);
                }
            });
        }
    }

    static addElement(k: string, flatSchema: Array<FlatElement>, tree: Array<string>) {
        let flatElement = flatSchema.find(f => f.name === k);
        if (!flatElement) {
            flatElement = new FlatElement();
            flatElement.name = k;
            flatElement.positions = new Array<FlatElementPosition>();
            flatSchema.push(flatElement);

        }
        let flatElementPosition = new FlatElementPosition();
        flatElementPosition.depth = tree.length;
        flatElementPosition.parent = tree;
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
}

export class FlatElementPosition {
    depth: number;
    parent: Array<string>;
    children: Array<string>;
}


