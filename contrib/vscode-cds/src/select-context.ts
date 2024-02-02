import { QuickPickItem, window } from "vscode";
import { Context } from "./lib/cds/models";
import { CDS } from "./lib/cds";

class ContextPickItem implements QuickPickItem {
    label: string;
    description: string;

    constructor(readonly context: Context) {
        this.label = context.context;
        this.description = context.host;
    }
}

export async function selectContext(): Promise<Context> {
    return new Promise<Context>(async (resolve, reject) => {
        const contexts = await CDS.getAvailableContexts();
        const input = window.createQuickPick<ContextPickItem>();

        input.placeholder = 'Select a context';
        input.items = contexts.map(c => new ContextPickItem(c));

        input.onDidChangeSelection(context => {
            input.hide();

            if (context) {
                resolve(context[0].context);
            }
        });

        input.show();
    });
}
