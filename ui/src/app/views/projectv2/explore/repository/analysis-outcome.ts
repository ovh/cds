import { DataEntity } from 'app/model/analysis.model';
import { ENTITY_MANAGE_PERMISSION, entityOfAnalysisFile } from './entities';

/** How an analysis went, as the Ant Design badge and step palettes name it. */
export type AnalysisOutcome = 'success' | 'warning' | 'error' | 'processing' | 'default';

/**
 * A Success that still reports an error is a partial one: some files were left out, for a
 * permission missing on their type; it reads as a warning rather than a plain success.
 */
export function analysisOutcome(status: string, error?: string): AnalysisOutcome {
    switch (status) {
        case 'Success': return error ? 'warning' : 'success';
        case 'Error': return 'error';
        case 'InProgress': return 'processing';
        default: return 'default';
    }
}

/** What became of a file of the analysis, in a few words; nothing for a registered one. */
export function analysisFileLabel(file: DataEntity): string {
    switch (file.status) {
        case 'Success': return '';
        case 'Skipped': {
            const permission = ENTITY_MANAGE_PERMISSION[entityOfAnalysisFile(file.path, file.file_name)?.type];
            return permission ? `skipped: missing permission ${permission}` : 'skipped: missing permission on this type';
        }
        default: return 'not processed';
    }
}
