import { DataEntity } from 'app/model/analysis.model';
import { analysisFileLabel, analysisOutcome } from './analysis-outcome';

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
        expect(analysisOutcome('Skipped', 'no file found')).toBe('default');
        expect(analysisOutcome(undefined)).toBe('default');
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
