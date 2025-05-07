import { QuickPickItem, QuickPickItemKind, ThemeIcon, window } from "vscode";

import { Project } from "../cds/models";
import { CDS } from "../cds";
import { Journal } from "../utils/journal";

class ProjectPickItem implements QuickPickItem {
    readonly iconPath?: ThemeIcon;
    readonly label: string;
    readonly description: string;
    readonly detail: string;

    constructor(readonly project?: Project, readonly kind = QuickPickItemKind.Default) {
        this.label = project?.key || '';
        this.description = project?.name || '';
        this.detail = project?.description || '';
    }
}

class ProjectPickItemSeparator extends ProjectPickItem {
    constructor() {
        super(undefined, QuickPickItemKind.Separator);
    }
}

export function selectProject(): Promise<Project> {
    return new Promise<Project>(async (resolve, reject) => {
        const input = window.createQuickPick<ProjectPickItem>();

        input.placeholder = 'Select a project';
        input.busy = true;

        input.onDidChangeSelection(project => {
            input.hide();

            if (project && project[0].project) {
                Journal.logInfo(`Selected project: ${JSON.stringify(project)}`);
                resolve({
                    ...project[0].project,
                    found: true,
                });
            }
        });

        CDS.getProjects().then(projects => {
            const items: ProjectPickItem[] = [...projects.map(p => new ProjectPickItem(p))];

            Journal.logInfo(JSON.stringify(items));

            input.items = items;
            input.busy = false;
        })

        input.show();
    });
}
