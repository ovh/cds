export const ActionTypeAscode = 'ActionAsCode';


export class ActionAsCode {
    name: string;
    inputs: { [k: string]: ActionInput };
}

export class ActionInput {
    description: string;
    default: string;
}
