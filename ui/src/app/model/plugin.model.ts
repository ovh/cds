
export class Plugin {
    name: string;
    type: string;
    inputs: {[key: string]: PluginInput};
}

export class PluginInput {
    type: string;
    description: string;
    advanced: boolean;
    default: string;
}
