import { Command } from ".";
import { selectContext } from "../forms/select-context";
import { CDS } from "../cds";
import { Journal } from "../utils/journal";
import { updateContext } from "../utils/context";

export const SetCurrentContextCommandID = 'vscode-cds.setCurrentContext';

export class SetCurrentContextCommand implements Command {
    getID(): string {
        return SetCurrentContextCommandID
    }

    async run(): Promise<void> {
        const context = await selectContext();
        try {
            await CDS.setCurrentContext(context.context);
            await updateContext();
        } catch (e) {
            Journal.logError(e as Error);
        }
    }
}
