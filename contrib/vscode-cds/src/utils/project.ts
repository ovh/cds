import { setProject } from "../events/project";
import { CDS } from "../cds";
import { Journal } from "./journal";

export async function updateProject(repository: string | null): Promise<void> {
    setProject(null);

    if (repository) {
        try {
            const project = await CDS.getCurrentProject();
            setProject(project);
        } catch (e) {
            Journal.logError(new Error(`Cannot get the current project: ${e}`));
            setProject(null);
        }
    }
}
