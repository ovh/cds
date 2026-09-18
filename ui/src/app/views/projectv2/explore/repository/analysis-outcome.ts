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
