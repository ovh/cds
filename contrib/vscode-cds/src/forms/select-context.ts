import { QuickPickItem, window } from "vscode";
import { Context } from "../cds/models";
import { CDS } from "../cds";

class ContextPickItem implements QuickPickItem {
    label: string;
    description: string;

    constructor(readonly context: Context) {
        this.label = context.context;
        this.description = context.host;
    }
}

export function selectContext(): Promise<Context> {
    return new Promise<Context>((resolve, reject) => {
        const input = window.createQuickPick<ContextPickItem>();

        input.busy = true;
        input.placeholder = 'Select a context';

        input.onDidChangeSelection(context => {
            input.hide();

            if (context) {
                resolve(context[0].context);
            }
        });

        CDS.getAvailableContexts().then(contexts => {
            input.items = contexts.map(c => new ContextPickItem(c));
            input.busy = false;
        });

        input.show();
    });
}
