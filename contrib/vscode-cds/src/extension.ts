import * as cp from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import * as vscode from 'vscode';

import { onContextChanged } from './events/context';
import { CDSWorkflowPreview } from "./preview";
import { Journal } from './utils/journal';
import { CDS } from './cds';
import { createContextStatusBarItem } from './components/context-status';
import { SetCurrentContext, registerCommand } from './commands';
import { updateContext } from './utils/context';
import { SetCurrentProjectCommand } from './commands/set-current-project';
import { updateProject } from './utils/project';
import { createProjectStatusBarItem } from './components/project-status';
import { Context } from './cds/models';
import { onGitRepositoryChanged, updateGitRepository } from './events/git-repository';
import { PreviewWorkflowCommand } from './commands/preview-workflow';
import { isCDSActionFile, isCDSWorkerModelFile, isCDSWorkflowFile } from './cds/file_utils';
import { ClearCacheCommand } from './commands/clear-cache';
import { WorkflowViewProvider, WorkflowItem, RunItem, RepoItem } from './workflowViewProvider';
import { onProjectChanged } from './events/project';

const CDS_SCHEMA = 'cds';

const schemas = {
    'action': [
        new RegExp(/\/.cds\/actions\/.+\.ya?ml$/),
    ],
    'job': [
        new RegExp(/\/.cds\/jobs\/.+\.ya?ml$/),
    ],
    'workflow': [
        new RegExp(/\/.cds\/workflows\/.+\.ya?ml$/),
    ],
    'worker-model': [
        new RegExp(/\/.cds\/worker-models\/.+\.ya?ml$/),
    ],
    'workflow-template': [
        new RegExp(/\/.cds\/workflow-templates\/.+\.ya?ml$/),
    ],
};

export async function activate(context: vscode.ExtensionContext) {
    const yamlExtension = vscode.extensions.getExtension('redhat.vscode-yaml');
    if (!yamlExtension) {
        vscode.window.showErrorMessage(
            'The "YAML Language Support by Red Hat" extension is required for the CDS extension to work properly. Please install it and reload the window.'
        );
        return;
    }
    const yamlExtensionAPI = await yamlExtension.activate();

    // instanciate the preview component
    const workflowPreview = new CDSWorkflowPreview(context);

    // register the commands
    registerCommand(context, new ClearCacheCommand());
    registerCommand(context, new SetCurrentContext());
    registerCommand(context, new SetCurrentProjectCommand());
    registerCommand(context, new PreviewWorkflowCommand(workflowPreview));

    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(event => {
        if (event.affectsConfiguration('cds.config')) {
            updateContext();
        }
    }));

    context.subscriptions.push(vscode.window.onDidChangeActiveTextEditor(editor => {
        updateGitRepository();
        updateVscodeContext(editor);
    }));


    // when the CDS context has changed
    context.subscriptions.push(onContextChanged(async (context) => {
        if (context) {
            updateSchema(context);
        }
    }));

    // when the current git repository has changed
    context.subscriptions.push(onGitRepositoryChanged(async (repository) => {
        updateProject(repository);
    }));

    // register the schema provider
    yamlExtensionAPI.registerContributor(CDS_SCHEMA, onRequestSchemaURI, onRequestSchemaContent);

    // creates the status bar displaying the current context
    createContextStatusBarItem(context);

    // creates the status bar displaying the current project
    createProjectStatusBarItem(context);

    // init the update of the context
    updateContext();

    // init the update of the repository
    updateGitRepository();

    // init the vscode context
    updateVscodeContext(vscode.window.activeTextEditor);

    // ── Workflow Explorer ────────────────────────────────────────────────────
    const workflowViewProvider = new WorkflowViewProvider();

    context.subscriptions.push(workflowViewProvider);
    context.subscriptions.push(
        vscode.window.registerTreeDataProvider('vscode-cds-workflows', workflowViewProvider)
    );

    context.subscriptions.push(
        vscode.workspace.onDidChangeWorkspaceFolders(() => workflowViewProvider.refresh())
    );

    // Refresh workflow view when CDS project or context changes
    context.subscriptions.push(onProjectChanged((project) => {
        workflowViewProvider.setProjectKey(project?.key);
    }));
    context.subscriptions.push(onContextChanged(() => workflowViewProvider.refresh()));

    // Initialize the workflow view with the current project (event may have fired before listener was registered)
    CDS.getCurrentProject().then((project) => {
        if (project) {
            workflowViewProvider.setProjectKey(project.key);
        }
    }).catch(() => { /* ignore */ });

    context.subscriptions.push(vscode.commands.registerCommand('vscode-cds.refreshWorkflowView', () => {
        workflowViewProvider.refresh();
    }));

    context.subscriptions.push(vscode.commands.registerCommand('vscode-cds.triggerWorkflow', async (item?: WorkflowItem | RepoItem) => {
        let projKey: string | undefined;
        let vcsName: string | undefined;
        let repoId: string | undefined;
        let workflowName: string | undefined;
        let repoRoot: string | undefined;

        if (item instanceof WorkflowItem && item.repo.cdsRepo) {
            projKey = item.repo.cdsRepo.projectKey;
            vcsName = item.repo.cdsRepo.vcsName;
            repoId = item.repo.cdsRepo.id;
            workflowName = item.cdsWorkflowName;
            repoRoot = item.repo.repoRoot;
        } else if (item instanceof RepoItem && item.cdsRepo) {
            projKey = item.cdsRepo.projectKey;
            vcsName = item.cdsRepo.vcsName;
            repoId = item.cdsRepo.id;
            repoRoot = item.repoRoot;
            workflowName = await vscode.window.showInputBox({
                title: 'Workflow name',
                prompt: 'Enter the CDS v2 workflow name (filename without .yaml)',
            });
            if (!workflowName) { return; }
        }

        if (!projKey || !vcsName || !repoId || !workflowName) {
            vscode.window.showErrorMessage('No CDS repository data associated with this item.');
            return;
        }

        const branch = await vscode.window.showInputBox({
            title: 'Branch',
            prompt: 'Branch to run on (leave empty for default)',
            value: await currentGitBranch(repoRoot),
        });
        if (branch === undefined) { return; }

        const cmd = CDS.buildTriggerV2Command(projKey, vcsName, repoId, workflowName, branch || undefined);
        const terminal = vscode.window.createTerminal({ name: `CDS run: ${workflowName}`, cwd: repoRoot });
        terminal.show();
        terminal.sendText(cmd);
    }));

    context.subscriptions.push(vscode.commands.registerCommand('vscode-cds.stopRun', async (item?: RunItem) => {
        if (!item?.run.id) { vscode.window.showErrorMessage('No run selected.'); return; }
        const confirmed = await vscode.window.showWarningMessage(
            `Stop run #${item.run.runNumber} of ${item.run.workflowName}?`, { modal: true }, 'Stop'
        );
        if (confirmed !== 'Stop') { return; }
        try {
            await CDS.stopRun(item.run.projectKey, item.run.id);
            vscode.window.showInformationMessage(`Run #${item.run.runNumber} stopped.`);
            workflowViewProvider.refreshRuns(item.workflow);
        } catch (e) {
            vscode.window.showErrorMessage(`Failed to stop run: ${(e as Error).message}`);
        }
    }));

    context.subscriptions.push(vscode.commands.registerCommand('vscode-cds.restartRun', async (item?: RunItem) => {
        if (!item?.run.id) { vscode.window.showErrorMessage('No run selected.'); return; }
        try {
            await CDS.restartRun(item.run.projectKey, item.run.id);
            vscode.window.showInformationMessage(`Run #${item.run.runNumber} restarted.`);
            workflowViewProvider.refreshRuns(item.workflow);
        } catch (e) {
            vscode.window.showErrorMessage(`Failed to restart run: ${(e as Error).message}`);
        }
    }));

    context.subscriptions.push(vscode.commands.registerCommand('vscode-cds.viewRunLogs', async (item?: RunItem) => {
        if (!item?.run.id) { vscode.window.showErrorMessage('No run selected.'); return; }
        const repoRoot = item.workflow.repo.repoRoot;
        const logsDir = repoRoot ? path.join(repoRoot, '.logs') : undefined;
        if (logsDir) {
            await fs.promises.mkdir(logsDir, { recursive: true });
        }
        const cmd = CDS.buildLogsCommand(item.run.projectKey, item.run.id);
        const terminal = vscode.window.createTerminal({
            name: `CDS logs: #${item.run.runNumber} ${item.run.workflowName}`,
            cwd: logsDir ?? repoRoot,
        });
        terminal.show();
        terminal.sendText(cmd);
    }));

    context.subscriptions.push(vscode.commands.registerCommand('vscode-cds.refreshRuns', (item?: WorkflowItem) => {
        if (item) { workflowViewProvider.refreshRuns(item); }
    }));
}

async function currentGitBranch(cwd?: string): Promise<string> {
    return new Promise((resolve) => {
        cp.exec('git rev-parse --abbrev-ref HEAD', { cwd }, (_err, stdout) => {
            resolve(stdout.trim() || '');
        });
    });
}

async function updateSchema(context: Context) {
    Journal.logInfo(`Downloading schema for "${context.context}"...`);
    await CDS.downloadSchemas();
    Journal.logInfo(`Downloaded schema for "${context.context}"`);
}

function onRequestSchemaURI(resource: string): string | undefined {
    for (const [type, patterns] of Object.entries(schemas)) {
        for (const pattern of patterns) {
            if (pattern.test(resource)) {
                return `${CDS_SCHEMA}:${type}`;
            }
        }
    }
    return undefined;
}

function onRequestSchemaContent(schemaUri: string): string | undefined {
    const parsedUri = vscode.Uri.parse(schemaUri);

    if (parsedUri.scheme !== CDS_SCHEMA) {
        return undefined;
    }

    return getSchemaContent(parsedUri.path);
}

function getSchemaContent(name: string): string | undefined {
    const schemaPath = path.join(os.homedir(), '.cds-schema', name + '.v2.schema.json');
    try {
        return fs.readFileSync(schemaPath, 'utf-8');
    } catch {
        return undefined;
    }
}

function updateVscodeContext(editor?: vscode.TextEditor) {
    // set vscode context
    if (editor?.document && isCDSWorkflowFile(editor.document)) {
        vscode.commands.executeCommand('setContext', 'isCDSWorkflowFile', true);
    } else {
        vscode.commands.executeCommand('setContext', 'isCDSWorkflowFile', false);
    }

    if (editor?.document && isCDSActionFile(editor.document)) {
        vscode.commands.executeCommand('setContext', 'isCDSActionFile', true);
    } else {
        vscode.commands.executeCommand('setContext', 'isCDSActionFile', false);
    }

    if (editor?.document && isCDSWorkerModelFile(editor.document)) {
        vscode.commands.executeCommand('setContext', 'isCDSWorkerModelFile', true);
    } else {
        vscode.commands.executeCommand('setContext', 'isCDSWorkerModelFile', false);
    }
}
