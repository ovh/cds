import { HookEventWorkflowStatus, RepositoryHookEvent, RepositoryHookWorkflow, WorkflowHookEventName } from 'app/model/project.model';
import { cleanErrorMessage, eventAuthor, eventAuthorKnown, hintFor, hookEventVerdict } from './hook-event-verdict';

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
        expect(v.steps[3].status).toBe('pending');
    });

    it('blames the analysis when the one of the project failed', () => {
        const v = hookEventVerdict(event({ status: 'Error', analyses: [{ analyze_id: 'a1', status: 'Error', project_key: PROJECT }] }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('Analysis failed');
        expect(v.detail).toBe('Workflows of this ref were not evaluated');
        expect(v.steps[2].status).toBe('error');
        expect(v.steps[3].status).toBe('wait');
    });

    it('names the failure of the analysis when the event carries it', () => {
        const v = hookEventVerdict(event({ status: 'Error', analyses: [{ analyze_id: 'a1', status: 'Error', project_key: PROJECT, error: "workflow release: integration artifactory doesn't exist.\nsecond line" }] }), PROJECT, false);
        expect(v.detail).toBe("workflow release: integration artifactory doesn't exist.");
        expect(v.steps[2].description).toContain("workflow release: integration artifactory doesn't exist.");
    });

    it('warns on an analysis that succeeded but left files out', () => {
        const v = hookEventVerdict(event({ analyses: [{ analyze_id: 'a1', status: 'Success', project_key: PROJECT, error: "User doesn't have the permission to manage WorkerModel" }], workflows: [workflow('checkout', HookEventWorkflowStatus.Done, { run_number: '12' })] }), PROJECT, false);
        expect(v.level).toBe('success');
        expect(v.steps[2].status).toBe('warning');
        expect(v.steps[2].description).toBe("Success · User doesn't have the permission to manage WorkerModel");
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

    it('reads a skipped event with a reason as the error it is, listing the workflows it skipped', () => {
        const v = hookEventVerdict(event({ status: 'Skipped', username: null, initiator: null, sign_key: '4603FA40B0D5B3F7', last_error: 'User with key 4603FA40B0D5B3F7 not found',
            workflows: [workflow('distant1', HookEventWorkflowStatus.Skipped, { error: 'User with key 4603FA40B0D5B3F7 not found' })] }), PROJECT, true);
        expect(v.level).toBe('error');
        expect(v.label).toBe('Signer unknown');
        expect(v.steps[1].status).toBe('error');
        expect(v.steps[3].description).toContain('did not start: distant1');
    });

    it('names an unsigned commit for what it is', () => {
        const v = hookEventVerdict(event({ status: 'Skipped', username: null, last_error: 'Commit not signed' }), PROJECT, false);
        expect(v.label).toBe('Commit not signed');
        expect(v.hint).toContain('signed');
    });

    it('treats a workflow skipped by its filters as expected, not as a problem', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('fromrunChangeset', HookEventWorkflowStatus.Skipped, { error: 'no file matches path filters' })] }), PROJECT, false);
        expect(v.level).toBe('filtered');
        expect(v.label).toBe('1 workflow skipped by its filters');
        expect(v.detail).toBe('no file matches path filters');
        expect(v.steps[3].status).toBe('skipped');
        expect(v.steps[3].description).toContain('skipped: fromrunChangeset (no file matches path filters)');
    });

    it('reports a workflow skipped with a reason as not started', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('deploy', HookEventWorkflowStatus.Skipped, { error: 'unknown user' })] }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('1 workflow did not start');
        expect(v.detail).toBe('unknown user');
    });

    it('blames the workflows when the git information of the ref could not be read', () => {
        const v = hookEventVerdict(event({ status: 'Error', event_name: WorkflowHookEventName.WorkflowHookEventNamePullWorkflowRun,
            last_error: 'internal server error (caused by: unable to get git info: API Error: resource not found (request_id: f8af8612-0c99-4a84-be1b-5166d5f717cd))' }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('Git information unavailable');
        expect(v.detail).toBe('unable to get git info: resource not found');
        expect(v.steps[0].status).toBe('finish');
        expect(v.steps[3].status).toBe('error');
        expect(v.hint).toContain('vcs credentials');
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
        expect(v.label).toBe('1 of 2 did not start');
        expect(v.steps[3].status).toBe('warning');
        expect(v.hint).toContain('worker model');
    });

    it('is an error when every workflow failed to start', () => {
        const v = hookEventVerdict(event({ workflows: [workflow('deploy', HookEventWorkflowStatus.Error, { error: 'boom' })] }), PROJECT, false);
        expect(v.level).toBe('error');
        expect(v.label).toBe('1 workflow did not start');
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

describe('cleanErrorMessage', () => {
    it('unwraps the API envelope and drops the request id', () => {
        expect(cleanErrorMessage('internal server error (caused by: unable to get git info: API Error: resource not found (request_id: abc))')).toBe('unable to get git info: resource not found');
        expect(cleanErrorMessage('User with key 4603 not found')).toBe('User with key 4603 not found');
        expect(cleanErrorMessage('unable to run workflow checkout: forbidden (from: user forgejo_build/robot-cds-sgu has no right to trigger a workflow, request_id: bbee06db-3bfc)'))
            .toBe('unable to run workflow checkout: user forgejo_build/robot-cds-sgu has no right to trigger a workflow');
        expect(cleanErrorMessage('not found (from: entity go not found)')).toBe('entity go not found');
        expect(cleanErrorMessage(null)).toBe('');
    });
});

describe('hintFor', () => {
    it('knows the usual failures and stays quiet otherwise', () => {
        expect(hintFor('yaml: unmarshal errors: line 12')).toContain('definition file');
        expect(hintFor('user robot has no right to trigger a workflow')).toContain('trigger role');
        expect(hintFor('something else entirely')).toBeUndefined();
        expect(hintFor('')).toBeUndefined();
    });
});
