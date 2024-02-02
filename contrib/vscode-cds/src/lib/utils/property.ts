import { ConfigurationTarget, workspace } from "vscode";
import { LogCategory } from "./journal";

interface IPropertiesMap {
    "logLevel": LogCategory[];
    "config": string;
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

    public static getConfigFileName(configFile: string) {
        if (configFile.startsWith("~")) {
            const homedir = require('os').homedir();
            configFile = homedir + configFile.substring(1);
        }
        return configFile;
    }
}
