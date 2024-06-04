import { exec } from "child_process";
import { window, workspace } from "vscode";

import { Journal } from "../utils/journal";
import { isActiveEditorValid } from "../utils/editor";
import { Property } from "../utils/property";
import { Context, Project } from "./models";
import { getGitLocalConfig, getGitRepositoryPath, setGitLocalConfig } from "../utils/git";
import { Cache } from "../utils/cache";
import { WorkflowGenerateRequest, WorkflowGenerateResponse } from "./models/WorkflowGenerated";

const defaultConfigFile = '~/.cdsrc';

const GIT_CONFIG_SECTION = "cds";
const GIT_CONFIG_PROJECT = "project";

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

    static async generateWorkflowFromTemplate(req: WorkflowGenerateRequest): Promise<WorkflowGenerateResponse> {
        let args: string[] = [];
        args.push("X", "template", "generate-from-file", req.filePath);
        Object.keys(req.params).forEach(k => {
            args.push("-p", k+"="+req.params[k]);
        });
        args.push("--format", "json");

        const resp = (await CDS.getInstance().runCtl(...args));
        const generatedWorkflow = JSON.parse(resp);
        return generatedWorkflow;
    }


    static async getProjects(): Promise<Project[]> {
        const context = await CDS.getCurrentContext();

        if (context) {
            const cachedProjects = Cache.get<Project[]>(`${context.context}.projects`);

            if (cachedProjects) {
                return cachedProjects;
            }
        }

        const projectsJson = (await CDS.getInstance().runCtl("project", "list", "--format", "json"));
        const projects = JSON.parse(projectsJson);

        if (context) {
            Cache.set(`${context.context}.projects`, projects, Cache.TTL_HOUR * 24);
        }

        return projects;
    }

    static async getCurrentProject(): Promise<Project | null> {
        if (!window.activeTextEditor || window.activeTextEditor.document.uri.scheme !== 'file') {
            return null;
        }

        try {
            const repository = await getGitRepositoryPath(window.activeTextEditor.document.fileName);
            const projectKey = await getGitLocalConfig(repository, GIT_CONFIG_SECTION, "project");

            if (!projectKey) {
                return null;
            }

            const foundProject = (await CDS.getProjects()).filter(p => p.key === projectKey)[0] ?? null;

            if (!foundProject) {
                return {
                    key: projectKey,
                    name: projectKey,
                    description: '',
                    favorite: 'false',
                    found: false,
                };
            }

            return {
                ...foundProject,
                found: true,
            }
        } catch (e) {
            return null;
        }
    }

    static async setCurrentProject(project: Project): Promise<void> {
        if (!window.activeTextEditor || window.activeTextEditor.document.uri.scheme !== 'file') {
            return;
        }

        if (project) {
            const repository = await getGitRepositoryPath(window.activeTextEditor.document.fileName);
            await setGitLocalConfig(repository, GIT_CONFIG_SECTION, GIT_CONFIG_PROJECT, project.key);
        }
    }

    static async downloadSchemas(): Promise<void> {
        await CDS.getInstance().runCtl("tools", "yaml-schema", "vscode");
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
