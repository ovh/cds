import { setContext } from "../events/context";
import { CDS } from "../cds";
import { Journal } from "./journal";

export async function updateContext(): Promise<void> {
    try {
        const context = await CDS.getCurrentContext();
        setContext(context);
    } catch (e) {
        Journal.logError(new Error(`Cannot get the current context: ${e}`));
        setContext(null);
    }
}
