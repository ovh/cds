import { exec } from "child_process";
import { window, workspace } from "vscode";
import { isActiveEditorValid } from "./util.editor";
import { Journal } from "./util.journal";
import { Property } from "./util.property";

export class CdsCtl {
    private currentProject: any;
    private currentWorkflow: any;

    private configFile: string;
    private contextName: string;
    private configUiURL: string;
    private initialized: boolean;

    constructor(configFile: string, contextName: string) {
        this.configFile = configFile;
        this.contextName = contextName;
        this.configUiURL = "...";
        this.initialized = false;
    }

    public getContextName(): string {
        return this.contextName;
    }

    public getConfigUiURL(): string {
        return this.configUiURL;
    }

    public getConfigFile(): string {
        return this.configFile;
    }

    public async init(): Promise<void> {
        try {
            const rawCmd = this.buildRawCDSCommand("admin curl /config/user");
            const configUser = await <Promise<any>>this.runCommand(rawCmd);
            this.configUiURL = configUser["url.ui"];
            this.initialized = true;
            await this.runCommand(this.buildCDSCommand("user me"));
        } catch (e) {
            window.showErrorMessage(`Unable to initialize context: ${e}. You should run this command outside vscode to investigate.`);
        }
    }

    public async runCdsCommand(cmd: string): Promise<any> {
        if (!this.initialized) {
            await this.init();
        }
        return this.runCommand(this.buildCDSCommand(cmd)).then(
            (data) => {
                return new Promise((resolve, reject) => {
                    resolve(JSON.parse(data));
                });
            },
        );
    }

    public async getCDSProject(): Promise<any> {
        return this.runCommand("git config --local cds.project").then(
            async () => {
                const data = await this.runCommand(this.buildCDSCommand("project show"));
                return new Promise((resolve, reject) => {
                    const proj = JSON.parse(data);
                    this.currentProject = proj;
                    resolve(proj);
                });
            },
        );
    }

    public async getCDSWorkflow(): Promise<any> {
        return this.runCommand("git config --local cds.workflow").then(
            async () => {
                const data = await this.runCommand(this.buildCDSCommand("workflow show"));
                return new Promise((resolve, reject) => {
                    const w = JSON.parse(data);
                    this.currentWorkflow = w;
                    resolve(w);
                });
            },
        );
    }

    public async getCDSWorkflowStatus(): Promise<any> {
        return this.runCommand("git config --local cds.workflow").then(
            async () => {
                const data = await this.runCommand(this.buildCDSCommand("workflow status"));
                return new Promise((resolve, reject) => {
                    const w = JSON.parse(data);
                    resolve(w);
                });
            },
        );
    }

    public async workflowInfo(): Promise<void> {
        try {
            await window.showInformationMessage(`${this.currentProject.key}/${this.currentWorkflow.name}`);
        } catch (e) {
            Journal.logError(e);
        }
    }

    public async runCommand(cmd: string): Promise<any> {
        const pwd = this.getCurrentPath();
        Journal.logInfo(`running command ${cmd} from directory ${pwd}`);
        return new Promise((resolve, reject) => {
            exec(cmd,
                {
                    cwd: pwd,
                },
                (error, stdout, stderr) => {
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

    public buildRawCDSCommand(cmd: string): string {
        const rootCmd = Property.get("binaryFileLocation") || "cdsctl";
        const configFile = this.configFile || "~/.cdsrc";
        return `${rootCmd} -f ${configFile} -c ${this.contextName} ${cmd}`;
    }

    private buildCDSCommand(cmd: string): string {
        const rootCmd = this.buildRawCDSCommand(cmd);
        return `${rootCmd} --format json`;
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
