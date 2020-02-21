import { workspace, ConfigurationTarget } from "vscode";
import { LogCategory } from "./util.journal";

interface IPropertiesMap {
    "binaryFileLocation": string;
    "logLevel": LogCategory[];
    "progressSpinner": string[];
    "statusBarPositionPriority": number;
    "knownCdsconfigs": string[];
}

export class Property {
    public static get<K extends keyof IPropertiesMap>(name: K): IPropertiesMap[K] | undefined {
        const properties = workspace.getConfiguration("cds");
        return properties.get(name);
    }

    public static set<K extends keyof IPropertiesMap>(name: K, value: IPropertiesMap[K]) {
        const properties = workspace.getConfiguration("cds");
        properties.update(name, value, ConfigurationTarget.Global);
    }

    // delete deletes a value from an array
    public static delete<K extends keyof IPropertiesMap>(name: K, value: string) {
        const v = Property.get(name) as Array<string>;
        const index = v.indexOf(value, 0);
        if (index > -1) {
            v.splice(index, 1);
        }
        Property.set(name, v as never);
    }
}
