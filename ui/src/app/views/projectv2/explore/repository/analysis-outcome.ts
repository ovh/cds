import { DataEntity, RepositoryAnalysis } from 'app/model/analysis.model';
import { EntityType } from 'app/model/entity.model';
import { ENTITY_MANAGE_PERMISSION, ENTITY_TYPE_NOUNS, ENTITY_TYPE_ORDER, entityOfAnalysisFile } from './entities';

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

export type FileFate = 'processed' | 'skipped' | 'not processed';

/** One count of the summary of an analysis, coloured after what became of the files. */
export interface AnalysisSummaryPart {
    text: string;
    fate?: FileFate;
}

const FILE_FATES: Array<FileFate> = ['processed', 'skipped', 'not processed'];

function fileFate(file: DataEntity): FileFate {
    switch (file.status) {
        case 'Success': return 'processed';
        case 'Skipped': return 'skipped';
        default: return 'not processed';
    }
}

/** What the analysis did, counted by type: `4 workflows processed`, `1 action skipped`. */
export function analysisSummary(analysis: RepositoryAnalysis): Array<AnalysisSummaryPart> {
    const files = analysis.data?.entities ?? [];
    if (!files.length) {
        return [{ text: 'No file was processed' }];
    }
    const counts = new Map<string, number>();
    for (const file of files) {
        const key = `${entityOfAnalysisFile(file.path, file.file_name)?.type ?? ''}|${fileFate(file)}`;
        counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    const parts: Array<AnalysisSummaryPart> = [];
    for (const type of [...ENTITY_TYPE_ORDER, null]) {
        for (const fate of FILE_FATES) {
            const n = counts.get(`${type ?? ''}|${fate}`);
            if (n) {
                parts.push({ text: `${n} ${noun(type, n)} ${fate}`, fate });
            }
        }
    }
    return parts;
}

function noun(type: EntityType, n: number): string {
    const nouns = type ? ENTITY_TYPE_NOUNS[type] : ['file', 'files'];
    return nouns[n > 1 ? 1 : 0];
}
