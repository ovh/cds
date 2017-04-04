export class Parameter {
    id: number;
    name: string;
    type: string;
    value: string;
    description: string;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;
}
