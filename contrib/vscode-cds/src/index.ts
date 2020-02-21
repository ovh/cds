import { commands, env, ExtensionContext, MessageItem, window, workspace, Uri } from "vscode";
import { CDSContext, CDSObject, CDSResource, createExplorer, refreshExplorer } from "./explorer";
import { Property } from "./util.property";
import { CDSExt } from "./cdsext";
import { StatusBarView } from "./view.statusbar";
import { Journal } from "./util.journal";

export function activate(context: ExtensionContext) {
    const subscriptions = [
        commands.registerCommand("extension.vsCdsAddNewConfig", addNewConfig),
        commands.registerCommand('extension.vsCdsRemoveConfigFile', vsCdsremoveConfigFile),
        commands.registerCommand('extension.vsCdsSetAsCurrentContext', vsCdsSetAsCurrentContext),
        commands.registerCommand('extension.vsCdsOpenBrowserWorkflow', vsCdsOpenBrowserNode),
        commands.registerCommand('extension.vsCdsOpenBrowserWorkflowRun', vsCdsOpenBrowserNode),
        commands.registerCommand('extension.vsCdsOpenBrowserWorkflowStatusBar', vsCdsOpenBrowserStatusBar),
        commands.registerCommand('extension.vsCdsOpenBrowserProject', vsCdsOpenBrowserNode),
        commands.registerCommand('extension.vsCdsOpenBrowserApplication', vsCdsOpenBrowserNode),
        commands.registerCommand('extension.vsCdsOpenBrowserPipeline', vsCdsOpenBrowserNode),
        commands.registerCommand('extension.vsCdsShowStepLogs', vsCdsShowStepLogs),
    ];
    subscriptions.forEach((element) => {
        context.subscriptions.push(element);
    });

    const treeProvider = createExplorer();
    commands.registerCommand("extension.vsCdsRefreshExplorer", () => treeProvider.refresh()),
        window.registerTreeDataProvider("extension.vsCdsExplorer", treeProvider);
}

export function deactivate() { }

async function addNewConfig(cdsconfig?: string): Promise<void> {
    const kc = await getCdsconfigSelection(cdsconfig);
    if (!kc) {
        return;
    }
    return undefined;
}

async function getCdsconfigSelection(cdsconfig?: string): Promise<string | undefined> {
    const addNewCDSConfigFile = "+ Add new cds config file";

    if (cdsconfig) {
        return cdsconfig;
    }
    const knownCdsconfigs = Property.get("knownCdsconfigs") || [];
    const picks = [addNewCDSConfigFile, ...knownCdsconfigs!];
    const pick = await window.showQuickPick(picks);

    if (pick === addNewCDSConfigFile) {
        const cdsconfigUris = await window.showOpenDialog({});
        if (cdsconfigUris && cdsconfigUris.length === 1) {
            const cdsconfigPath = cdsconfigUris[0].fsPath;
            knownCdsconfigs.push(cdsconfigPath);
            Property.set("knownCdsconfigs", knownCdsconfigs);
            return cdsconfigPath;
        }
        return undefined;
    }

    return pick;
}

async function vsCdsremoveConfigFile(explorerNode: CDSObject) {
    if (!explorerNode || !explorerNode.metadata.cdsctl.configFile) {
        return;
    }
    const contextObj = explorerNode.metadata as CDSContext;
    const deleteCancel: MessageItem[] = [{title: "Delete"}, {title: "Cancel", isCloseAffordance: true}];
    const answer = await window.showWarningMessage(`Do you want to remove the configuration file '${contextObj.cdsctl.getContextName()}'?`, ...deleteCancel);
    if (!answer || answer.isCloseAffordance) {
        return;
    }
    if (CDSExt.getInstance().currentContext === contextObj) {
        CDSExt.getInstance().currentContext = undefined;
    }
    Property.delete("knownCdsconfigs", contextObj.cdsctl.getConfigFile());
    refreshExplorer();
}

async function vsCdsSetAsCurrentContext(explorerNode: CDSObject) {
    if (!explorerNode || !explorerNode.metadata.cdsctl.configFile) {
        return;
    }

    const yesNo: MessageItem[] = [{title: "Yes"}, {title: "No", isCloseAffordance: true}];
    const contextObj = explorerNode.metadata as CDSContext;
    const answer = await window.showInformationMessage(`Do you want to set '${contextObj.name}' as the current context?`, ...yesNo);
    if (!answer || answer.isCloseAffordance) {
        return;
    }
    CDSExt.getInstance().currentContext = contextObj;
    refreshExplorer();
}

async function vsCdsOpenBrowserNode(explorerNode: CDSObject): Promise<void> {
    const r = explorerNode as CDSResource;
    env.openExternal(r.uri());
}

async function vsCdsOpenBrowserStatusBar(): Promise<void> {
    const project = await CDSExt.getInstance().currentContext!.cdsctl.getCDSProject();
    const workflow = await CDSExt.getInstance().currentContext!.cdsctl.getCDSWorkflow();
    CDSExt.getInstance().currentContext!.cdsctl.getConfigUiURL().then(
        (uiUri) => {
            const uri = Uri.parse(`${uiUri}/project/${project.key}/workflow/${workflow.name}`);
            env.openExternal(uri);
        }
    );
}

async function vsCdsShowStepLogs(explorerNode: CDSObject): Promise<void> {
    const r = explorerNode as CDSResource;

    const document = await workspace.openTextDocument({
        language: "plaintext",
        content: "TODO " + JSON.stringify(r),
    });
    window.showTextDocument(document);
}
