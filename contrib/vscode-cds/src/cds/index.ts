import { exec } from "child_process";
import { window, workspace } from "vscode";

import { Journal } from "../utils/journal";
import { isActiveEditorValid } from "../utils/editor";
import { Property } from "../utils/property";
import { Context, Project, CdsRepository, CdsWorkflowRun } from "./models";
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
        try {
            return JSON.parse(stdout);
        } catch {
            Journal.logError(new Error(`getAvailableContexts: failed to parse JSON output`));
            return [];
        }
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
        try {
            return JSON.parse(resp);
        } catch {
            Journal.logError(new Error(`generateWorkflowFromTemplate: failed to parse JSON output`));
            throw new Error("Failed to parse workflow generation response");
        }
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
        let projects: Project[];
        try {
            projects = JSON.parse(projectsJson);
        } catch {
            Journal.logError(new Error(`getProjects: failed to parse JSON output`));
            return [];
        }

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

    // ── V2 Repository & Workflow methods ─────────────────────────────────

    /**
     * List CDS v2 repositories for a given project.
     */
    static async listRepositories(projectKey: string): Promise<CdsRepository[]> {
        const out = await CDS.getInstance().runCtl(
            "experimental", "project", "repository", "list", projectKey, "--format", "json",
        );
        let items: Record<string, string>[];
        try {
            items = JSON.parse(out);
        } catch {
            Journal.logError(new Error(`listRepositories: failed to parse JSON output`));
            return [];
        }
        return items
            .filter((r) => r["repoName"] || r["repo_name"])
            .map((r) => ({
                id: r["id"] ?? "",
                vcsName: r["vcsName"] ?? r["vcs_name"] ?? "",
                repoName: r["repoName"] ?? r["repo_name"] ?? "",
                projectKey,
            }));
    }

    /**
     * Get run history for a specific v2 workflow.
     */
    static async getWorkflowHistory(
        projectKey: string,
        vcsName: string,
        repoId: string,
        workflowName: string,
        limit = 15,
    ): Promise<CdsWorkflowRun[]> {
        const out = await CDS.getInstance().runCtl(
            "experimental", "workflow", "history",
            projectKey, vcsName, repoId, workflowName,
            "--format", "json",
        );
        let items: Record<string, any>[];
        try {
            items = JSON.parse(out);
        } catch {
            Journal.logError(new Error(`getWorkflowHistory: failed to parse JSON output`));
            return [];
        }
        return items.slice(0, limit).map((r) => ({
            id: r["id"] ?? "",
            runNumber: parseInt(r["run_number"] ?? r["runnumber"] ?? "0", 10),
            status: r["status"] ?? "",
            started: r["started"] ?? r["start"] ?? r["last_modified"] ?? "",
            workflowName: r["workflow_name"] ?? r["workflowname"] ?? workflowName,
            projectKey: r["project_key"] ?? r["projectkey"] ?? projectKey,
            username: r["user"] ?? r["username"] ?? "",
            ref: r["ref_name"] ?? "",
            commit: r["commit"] ?? "",
        }));
    }

    /**
     * Search v2 workflow runs in a project with optional filters.
     */
    static async searchWorkflows(
        projectKey: string,
        opts?: { workflow?: string; repository?: string; limit?: number },
    ): Promise<CdsWorkflowRun[]> {
        const limit = opts?.limit ?? 200;
        const args = ["experimental", "workflow", "search", "--project", projectKey, "--limit", String(limit), "--format", "json"];
        if (opts?.workflow) {
            args.push("--workflow", opts.workflow);
        }
        if (opts?.repository) {
            args.push("--repository", opts.repository);
        }
        const out = await CDS.getInstance().runCtl(...args);
        try {
            return JSON.parse(out);
        } catch {
            Journal.logError(new Error(`searchWorkflows: failed to parse JSON output`));
            return [];
        }
    }

    /**
     * Discover workflow names scoped to a specific repository.
     * Uses the repository filter to make a single API call.
     */
    static async discoverRepoWorkflowNames(
        projectKey: string,
        vcsName: string,
        repoName: string,
    ): Promise<string[]> {
        try {
            const items = await CDS.searchWorkflows(projectKey, {
                repository: `${vcsName}/${repoName}`,
                limit: 200,
            });
            const names = new Set<string>();
            for (const r of items) {
                const name = r.workflowName;
                if (name) { names.add(name); }
            }
            return [...names];
        } catch {
            return [];
        }
    }

    /** Stop a running v2 workflow run. */
    static async stopRun(projectKey: string, runId: string): Promise<void> {
        await CDS.getInstance().runCtl("experimental", "workflow", "stop", projectKey, runId);
    }

    /** Restart failed/stopped jobs in a v2 workflow run. */
    static async restartRun(projectKey: string, runId: string): Promise<void> {
        await CDS.getInstance().runCtl("experimental", "workflow", "restart", projectKey, runId);
    }

    /** Build the cdsctl command to trigger a v2 workflow run (to run in a terminal). */
    static buildTriggerV2Command(
        projectKey: string,
        vcsName: string,
        repoId: string,
        workflowName: string,
        branch?: string,
        tag?: string,
    ): string {
        const esc = (s: string) => `'${s.replace(/'/g, "'\\''")}'`;
        let cmd = `cdsctl -f ${esc(CDS.getConfigFile())} -n experimental workflow run ${esc(projectKey)} ${esc(vcsName)} ${esc(repoId)} ${esc(workflowName)}`;
        if (branch) { cmd += ` --branch ${esc(branch)}`; }
        if (tag) { cmd += ` --tag ${esc(tag)}`; }
        return cmd;
    }

    /** Build the command to download run logs (to run in a terminal). */
    static buildLogsCommand(projectKey: string, runId: string): string {
        return `cdsctl -f ${CDS.getConfigFile()} -n experimental workflow logs download ${projectKey} ${runId}`;
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
                    timeout: 30_000,
                },
                (error, stdout, stderr) => {
                    Journal.logInfo(stdout)
                    Journal.logInfo(stderr)


                    if (error) {
                        Journal.logError(error);
                        return reject(error);
                    }
                    if (stderr) {
                        return reject(stderr);
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
