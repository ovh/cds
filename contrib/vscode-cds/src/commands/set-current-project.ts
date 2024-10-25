import { Command } from ".";
import { selectProject } from "../forms/select-project";
import { CDS } from "../cds";
import { Journal } from "../utils/journal";
import { setProject } from "../events/project";

export const SetCurrentProjectCommandID = 'vscode-cds.setCurrentProject';

export class SetCurrentProjectCommand implements Command {
    getID(): string {
        return SetCurrentProjectCommandID
    }

    async run(): Promise<void> {
        const project = await selectProject();
        try {
            await CDS.setCurrentProject(project);
            setProject(project);
        } catch (e) {
            Journal.logError(e as Error);
        }
    }
}
