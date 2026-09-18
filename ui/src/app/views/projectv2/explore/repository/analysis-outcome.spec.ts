import { DataEntity, RepositoryAnalysis } from 'app/model/analysis.model';
import { analysisFileLabel, analysisOutcome, analysisSummary } from './analysis-outcome';

describe('analysisOutcome', () => {
    it('is a success when nothing went wrong', () => {
        expect(analysisOutcome('Success')).toBe('success');
        expect(analysisOutcome('Success', '')).toBe('success');
    });

    it('is a warning when a success still reports left out files', () => {
        expect(analysisOutcome('Success', "User doesn't have the permission to manage WorkerModel")).toBe('warning');
    });

    it('is an error or in progress from the status alone', () => {
        expect(analysisOutcome('Error', 'anything')).toBe('error');
        expect(analysisOutcome('InProgress')).toBe('processing');
    });

    it('is neutral for a skipped or unknown status', () => {
        expect(analysisOutcome('Skipped', 'no file found')).toBe('none');
        expect(analysisOutcome(undefined)).toBe('none');
    });
});

describe('analysisFileLabel', () => {
    const file = (path: string, fileName: string, status: string): DataEntity => <DataEntity>{ path, file_name: fileName, status };

    it('names the permission a skipped file needs', () => {
        expect(analysisFileLabel(file('.cds/workflow-templates/', 'helloworld.yaml', 'Skipped'))).toBe('skipped: missing permission manage-workflow-template');
        expect(analysisFileLabel(file('.cds/worker-models/', 'debian.yml', 'Skipped'))).toBe('skipped: missing permission manage-worker-model');
    });

    it('falls back to the type when the folder names no permission', () => {
        expect(analysisFileLabel(file('.cds/jobs/', 'build.yaml', 'Skipped'))).toBe('skipped: missing permission on this type');
    });

    it('says nothing for a registered file and little for a failed one', () => {
        expect(analysisFileLabel(file('.cds/workflows/', 'go.yaml', 'Success'))).toBe('');
        expect(analysisFileLabel(file('.cds/workflows/', 'go.yaml', ''))).toBe('not processed');
    });
});

describe('analysisSummary', () => {
    const file = (path: string, fileName: string, status: string): DataEntity => <DataEntity>{ path, file_name: fileName, status };
    const analysis = (entities: Array<DataEntity>): RepositoryAnalysis => <RepositoryAnalysis>{ data: { entities } };

    it('counts the files by type and fate, types in the order of the tabs', () => {
        const a = analysis([
            file('.cds/worker-models/', 'debian.yml', 'Success'),
            file('.cds/workflows/', 'go.yaml', 'Success'),
            file('.cds/actions/', 'lint.yaml', 'Skipped'),
            file('.cds/workflows/', 'build.yaml', 'Success'),
            file('.cds/workflows/', 'broken.yaml', '')
        ]);
        expect(analysisSummary(a)).toEqual([
            { text: '2 workflows processed', fate: 'processed' },
            { text: '1 workflow not processed', fate: 'not processed' },
            { text: '1 action skipped', fate: 'skipped' },
            { text: '1 worker model processed', fate: 'processed' }
        ]);
    });

    it('counts a file of an unknown folder as a file', () => {
        expect(analysisSummary(analysis([file('.cds/', 'README.md', 'Success'), file('.cds/', 'notes.md', 'Success')]))).toEqual([{ text: '2 files processed', fate: 'processed' }]);
    });

    it('says so, in no colour, when the analysis read no file', () => {
        expect(analysisSummary(analysis([]))).toEqual([{ text: 'No file was processed' }]);
        expect(analysisSummary(<RepositoryAnalysis>{})).toEqual([{ text: 'No file was processed' }]);
    });
});
