export class Schema {
    $defs: { [name: string]: Definition };
    $id: string;
    $ref: string;
    $schema: string;

    getDefinitionByRef(ref: string): Definition {
        const defKey = Object.keys(this.$defs).find(key => `#/$defs/${key}` === ref);
        if (!defKey) { return null; }
        return this.$defs[defKey];
    }
}

export class Definition {
    type: string;
    properties: { [name: string]: Property };
    additionalProperties: boolean;
    allOf: DefinitionAllOf[];
    required: string[];
}

export class DefinitionAllOf {
    if: {
        properties: {
            [key: string]: {
                const: string;
            }
        };
    };
    then: {
        properties: {
            [key: string]: {
                $ref: string;
            }
        };
    };
}

export class Property {
    description: string;
    order: string;
    type: string;
    disabled: boolean;
    minLength: number;
    enum: string[];
    pattern: string;
}
