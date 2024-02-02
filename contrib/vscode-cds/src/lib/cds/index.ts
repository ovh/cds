import { exec } from "child_process";
import { window, workspace } from "vscode";

import { Journal } from "../utils/journal";
import { isActiveEditorValid } from "../utils/editor";
import { Property } from "../utils/property";
import { Context } from "./models";

const defaultConfigFile = '~/.cdsrc';

export class CDS {
    static getConfigFile(): string {
        const cdsrc = Property.get('config') || defaultConfigFile;
        return Property.getConfigFileName(cdsrc);
    }

    static async getAvailableContexts(): Promise<Context[]> {
        const stdout = await CDS.getInstance().runCtl("context", "list", "--format", "json");
        return JSON.parse(stdout);
    }

    static async setCurrentContext(context: string): Promise<void> {
        await CDS.getInstance().runCtl("context", "set-current", context);
    }

    static async getCurrentContext(): Promise<Context | null> {
        const contextName = (await CDS.getInstance().runCtl("context", "current")).trimEnd();

        if (!contextName) {
            return null;
        }

        const foundContext = (await CDS.getAvailableContexts()).filter(c => c.context === contextName);

        if (!foundContext) {
            return null;
        }

        return foundContext[0];
    }

    static getInstance(): CDS {
        if (!CDS.instance) {
            CDS.instance = new CDS();
        }
        return CDS.instance;
    }

    private static instance: CDS;

    private constructor() { }

    private async runCtl(...args: string[]): Promise<string> {
        const pwd = this.getCurrentPath();
        const cmd = `cdsctl -f ${CDS.getConfigFile()} -n ${args.join(" ")}`;

        Journal.logInfo(`running command ${cmd} from directory ${pwd}`);

        return new Promise((resolve, reject) => {
            exec(cmd,
                {
                    cwd: pwd,
                },
                (error, stdout, stderr) => {
                    Journal.logInfo(stdout)
                    Journal.logInfo(stderr)


                    if (error) {
                        Journal.logError(error);
                        reject(error);
                    }
                    if (stderr) {
                        reject(stderr);
                    }
                    resolve(stdout);
                });
        });
    }

    private getCurrentPath(): string {
        if (
            isActiveEditorValid()
            && window
            && window.activeTextEditor
        ) {
            const folder = workspace.getWorkspaceFolder(window.activeTextEditor.document.uri);
            return folder ? folder.uri.fsPath : "";
        }
        return "";
    }
}
