import { analysisOutcome } from './analysis-outcome';

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
