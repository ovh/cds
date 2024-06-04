import { Command } from ".";
import { Cache } from "../utils/cache";

export const ClearCacheCommandID = 'vscode-cds.clearCache';

export class ClearCacheCommand implements Command {
    getID(): string {
        return ClearCacheCommandID;
    }

    async run(): Promise<void> {
        Cache.clear();
    }
}
