import { HookEventWorkflowStatus, RepositoryHookEvent, RepositoryHookWorkflow, WorkflowHookEventName } from 'app/model/project.model';
import { eventAuthor, eventAuthorKnown, hintFor, hookEventVerdict } from './hook-event-verdict';

const PROJECT = 'PROJ';

const event = (overrides: Partial<RepositoryHookEvent>): RepositoryHookEvent => <RepositoryHookEvent>{
    uuid: 'ebed3a70', status: 'Done', event_name: WorkflowHookEventName.WorkflowHookEventNamePush, username: 'sguiheux',
    extracted_data: { ref: 'refs/heads/master', commit: '3f11f6fcd5abcdef' }, analyses: [], workflows: [],
    ...overrides
};
const workflow = (name: string, status: HookEventWorkflowStatus, extra: Partial<RepositoryHookWorkflow> = {}): RepositoryHookWorkflow =>
    <RepositoryHookWorkflow>{ project_key: PROJECT, workflow_name: name, status, ...extra };

describe('hookEventVerdict', () => {
    it('reports an event still being handled as in progress', () => {
        const v = hookEventVerdict(event({ status: 'Analyzing' }), PROJECT, false);
        expect(v.level).toBe('processing');
        expect(v.steps[2].status).toBe('process');
    });

    it('blames the analysis when the one of the project failed', () => {
        const v = hookEventVerdict(event({ status: 'Error', analyses: [{ analyze_id: 'a1', status: 'Error', project_key: PROJECT }] }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('Analysis failed');
        expect(v.steps[2].status).toBe('error');
        expect(v.steps[3].status).toBe('wait');
    });

    it('ignores the analyses of other projects', () => {
        const v = hookEventVerdict(event({ analyses: [{ analyze_id: 'a1', status: 'Error', project_key: 'OTHER' }], workflows: [workflow('checkout', HookEventWorkflowStatus.Done, { run_number: '12' })] }), PROJECT, false);
        expect(v.level).toBe('success');
    });

    it('blames the author when the signing key is unknown', () => {
        const v = hookEventVerdict(event({ status: 'Error', username: null, last_error: 'User with key 4603FA40 not found' }), PROJECT, false);
        expect(v.label).toBe('Signer unknown');
        expect(v.steps[1].status).toBe('error');
        expect(v.hint).toContain('signing key');
    });

    it('blames the reception for any other error', () => {
        const v = hookEventVerdict(event({ status: 'Error', last_error: 'unable to read hook payload\nsecond line' }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('unable to read hook payload');
        expect(v.steps[0].status).toBe('error');
    });

    it('warns when some workflows failed to start while others ran', () => {
        const v = hookEventVerdict(event({ workflows: [
            workflow('checkout', HookEventWorkflowStatus.Done, { run_number: '128' }),
            workflow('deploy', HookEventWorkflowStatus.Error, { error: 'no worker model "debian13"' })
        ] }), PROJECT, false);
        expect(v.level).toBe('warning');
        expect(v.label).toBe('1 of 2 failed to start');
        expect(v.hint).toContain('worker model');
    });

    it('is an error when every workflow failed to start', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('deploy', HookEventWorkflowStatus.Error, { error: 'boom' })] }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('1 workflow failed to start');
    });

    it('is processing while workflows are scheduled', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('a', HookEventWorkflowStatus.Scheduled), workflow('b', HookEventWorkflowStatus.Scheduled)] }), PROJECT, false);
        expect(v.level).toBe('processing');
        expect(v.label).toBe('2 workflows scheduled');
    });

    it('counts the triggered workflows of the project only', () => {
        const v = hookEventVerdict(event({ workflows: [
            workflow('checkout', HookEventWorkflowStatus.Done, { run_number: '128' }),
            <RepositoryHookWorkflow>{ project_key: 'OTHER', workflow_name: 'elsewhere', status: HookEventWorkflowStatus.Done }
        ] }), PROJECT, false);
        expect(v.level).toBe('success');
        expect(v.label).toBe('1 workflow triggered');
        expect(v.detail).toBe('checkout #128');
    });

    it('says when nothing matched', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('a', HookEventWorkflowStatus.Skipped)] }), PROJECT, false);
        expect(v.level).toBe('skipped');
        expect(v.label).toBe('No workflow matched');
        expect(v.detail).toContain('1 workflow evaluated');
    });

    it('never expects an analysis on a listened repository', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('distant1', HookEventWorkflowStatus.Done, { run_number: '12' })] }), PROJECT, true);
        expect(v.steps[2].status).toBe('wait');
        expect(v.steps[2].description).toContain('Not applicable');
    });

    it('describes the event with its kind, ref and short commit', () => {
        const v = hookEventVerdict(event({ event_type: 'opened', event_name: WorkflowHookEventName.WorkflowHookEventNamePullRequest }), PROJECT, false);
        expect(v.steps[0].description).toBe('pull-request (opened) on refs/heads/master · 3f11f6f');
    });
});

describe('eventAuthor', () => {
    it('prefers the CDS user of the initiator, then its vcs user, then the legacy field', () => {
        expect(eventAuthor(event({ initiator: <any>{ user: { username: 'sguiheux' }, vcs_username: 'steven' } }), PROJECT)).toBe('sguiheux');
        expect(eventAuthor(event({ initiator: <any>{ vcs_username: 'steven' }, username: null }), PROJECT)).toBe('steven');
        expect(eventAuthor(event({ initiator: null, username: 'legacy' }), PROJECT)).toBe('legacy');
        expect(eventAuthor(event({ initiator: null, username: null }), PROJECT)).toBeNull();
    });

    it('falls back to the owner of a workflow of the project, as a scheduler names nobody on the event', () => {
        const e = event({ initiator: null, username: null, event_name: WorkflowHookEventName.WorkflowHookEventNameScheduler, workflows: [
            <RepositoryHookWorkflow>{ project_key: 'OTHER', workflow_name: 'elsewhere', status: HookEventWorkflowStatus.Done, initiator: <any>{ user: { username: 'stranger' } } },
            workflow('nightly', HookEventWorkflowStatus.Done, { run_number: '4', initiator: <any>{ user_id: 'u1', user: { username: 'owner' } } })
        ] });
        expect(eventAuthor(e, PROJECT)).toBe('owner');
        expect(eventAuthorKnown(e, PROJECT)).toBeTrue();
    });

    it('counts a bare user id as identified, without a name to show', () => {
        const e = event({ initiator: null, username: null, workflows: [workflow('nightly', HookEventWorkflowStatus.Done, { initiator: <any>{ user_id: 'u1', user: {} } })] });
        expect(eventAuthorKnown(e, PROJECT)).toBeTrue();
        expect(eventAuthor(e, PROJECT)).toBeNull();
        expect(hookEventVerdict(e, PROJECT, false).steps[1].description).toBe('Identified');
    });

    it('identifies the author of an event from its initiator', () => {
        const v = hookEventVerdict(event({ username: null, initiator: <any>{ user_id: 'u1', user: { username: 'sguiheux' } }, sign_key: 'B3F1' }), PROJECT, false);
        expect(v.steps[1].status).toBe('finish');
        expect(v.steps[1].description).toBe('sguiheux · key B3F1');
    });
});

describe('hintFor', () => {
    it('knows the usual failures and stays quiet otherwise', () => {
        expect(hintFor('yaml: unmarshal errors: line 12')).toContain('definition file');
        expect(hintFor('something else entirely')).toBeUndefined();
        expect(hintFor('')).toBeUndefined();
    });
});
