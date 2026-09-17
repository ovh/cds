import { HookEventWorkflowStatus, RepositoryHookEvent, RepositoryHookWorkflow } from 'app/model/project.model';
import { Initiator } from 'app/model/analysis.model';

export type VerdictLevel = 'success' | 'error' | 'warning' | 'skipped' | 'processing';
export type StepStatus = 'finish' | 'error' | 'wait' | 'process';

export interface VerdictStep {
    title: string;
    status: StepStatus;
    description: string;
}

/** What happened to a repository event, in one line, and step by step. */
export interface HookEventVerdict {
    level: VerdictLevel;
    label: string;
    detail: string;
    /** Received, author identified, analysis, workflows: always these four, in this order. */
    steps: Array<VerdictStep>;
    /** A way out of the failure, when the message is one the table below knows. */
    hint?: string;
}

// Statuses the hooks service goes through before an event is settled
const IN_PROGRESS = ['Scheduled', 'Analyzing', 'CheckAnalyzing', 'WorkflowHooks', 'GitInfo', 'Workflow'];
const ANALYZING = ['Analyzing', 'CheckAnalyzing'];
const TRIGGERING = ['WorkflowHooks', 'GitInfo', 'Workflow'];

const HINTS: Array<{ pattern: RegExp, hint: string }> = [
    { pattern: /User with key/i, hint: 'The commit signing key is not linked to any CDS user: the author must add it to their CDS profile, then push again.' },
    { pattern: /worker model/i, hint: 'The workflow references a worker model that does not exist on this ref: add it under .cds/worker-models or fix its runs-on value.' },
    { pattern: /yaml|unmarshal|unknown field|cannot parse/i, hint: 'A definition file is invalid: fix it on this ref and push again, or trigger an analysis after amending.' }
];

/**
 * Whether an initiator names someone, by the rule the hooks service applies: a CDS user id or a vcs
 * user is enough, a name to show is not required.
 */
function initiatorKnown(initiator: Initiator): boolean {
    return !!initiator && (!!initiator.user_id || !!initiator.vcs_username);
}

function initiatorName(initiator: Initiator): string {
    return initiator?.user?.username || initiator?.vcs_username || null;
}

/** The initiators an event may be attributed to: its own, then the ones of the workflows of the project. */
function initiators(event: RepositoryHookEvent, projectKey: string): Array<Initiator> {
    const ofWorkflows = (event.workflows ?? []).filter(w => w.project_key === projectKey).map(w => w.initiator);
    return [event.initiator, ...ofWorkflows].filter(i => !!i);
}

/**
 * Who an event is attributed to. A push names its committer on the event; a scheduler, a workflow-run
 * or a manual hook names the owner of each workflow instead. Older events only carry a username.
 */
export function eventAuthor(event: RepositoryHookEvent, projectKey: string): string {
    return initiators(event, projectKey).map(initiatorName).find(n => !!n) || event.username || null;
}

export function eventAuthorKnown(event: RepositoryHookEvent, projectKey: string): boolean {
    return initiators(event, projectKey).some(initiatorKnown) || !!event.username;
}

export function hookEventVerdict(event: RepositoryHookEvent, projectKey: string, distant: boolean): HookEventVerdict {
    const workflows = (event.workflows ?? []).filter(w => w.project_key === projectKey);
    const done = workflows.filter(w => w.status === HookEventWorkflowStatus.Done);
    const failed = workflows.filter(w => w.status === HookEventWorkflowStatus.Error);
    const scheduled = workflows.filter(w => w.status === HookEventWorkflowStatus.Scheduled);
    const skipped = workflows.filter(w => w.status === HookEventWorkflowStatus.Skipped);
    const analysis = (event.analyses ?? []).find(a => a.project_key === projectKey);
    const inProgress = IN_PROGRESS.indexOf(event.status) !== -1;
    const lastError = (event.last_error ?? '').trim();

    const authorName = eventAuthor(event, projectKey);
    const authorKnown = eventAuthorKnown(event, projectKey);
    const received: VerdictStep = { title: 'Event received', status: 'finish', description: describeEvent(event) };
    const author: VerdictStep = {
        title: 'Author identified',
        status: authorKnown ? 'finish' : inProgress ? 'process' : 'wait',
        description: !authorKnown ? 'Not identified'
            : !authorName ? 'Identified'
            : event.sign_key ? `${authorName} · key ${event.sign_key}` : authorName
    };
    const analysisStep: VerdictStep = distant
        ? { title: 'Analysis', status: 'wait', description: 'Not applicable: the repository is not declared in this project.' }
        : analysis
            ? { title: 'Analysis', status: analysisStatus(analysis.status), description: analysis.status }
            : { title: 'Analysis', status: inProgress && ANALYZING.indexOf(event.status) !== -1 ? 'process' : 'wait', description: inProgress ? 'Pending' : 'Not run for this event' };
    const workflowsStep: VerdictStep = { title: 'Workflows', status: 'wait', description: describeWorkflows(done, failed, scheduled, skipped) };
    const steps = [received, author, analysisStep, workflowsStep];

    if (inProgress) {
        if (TRIGGERING.indexOf(event.status) !== -1) {
            workflowsStep.status = 'process';
        }
        return { level: 'processing', label: 'In progress', detail: event.status, steps };
    }

    if (analysis?.status === 'Error') {
        analysisStep.status = 'error';
        workflowsStep.description = 'Not evaluated: the analysis must succeed first.';
        return { level: 'error', label: 'Analysis failed', detail: 'Workflows of this ref were not evaluated', steps, hint: hintFor(lastError) };
    }

    if (event.status === 'Error' && !analysis) {
        if (/^User with key/i.test(lastError)) {
            author.status = 'error';
            author.description = lastError;
            workflowsStep.description = 'Not evaluated: the author must be identified first.';
            return { level: 'error', label: 'Signer unknown', detail: lastError, steps, hint: hintFor(lastError) };
        }
        received.status = 'error';
        received.description = `${received.description} · ${lastError}`;
        return { level: 'error', label: firstLine(lastError) || 'Error', detail: lastError, steps, hint: hintFor(lastError) };
    }

    if (failed.length > 0) {
        workflowsStep.status = 'error';
        const errors = failed.map(w => w.error).filter(e => !!e).join(' · ');
        const label = done.length > 0
            ? `${failed.length} of ${failed.length + done.length} failed to start`
            : `${failed.length} ${plural(failed.length, 'workflow')} failed to start`;
        return { level: done.length > 0 ? 'warning' : 'error', label, detail: errors, steps, hint: hintFor(errors) };
    }

    if (scheduled.length > 0) {
        workflowsStep.status = 'process';
        return { level: 'processing', label: `${scheduled.length} ${plural(scheduled.length, 'workflow')} scheduled`, detail: names(scheduled), steps };
    }

    if (done.length > 0) {
        workflowsStep.status = 'finish';
        return { level: 'success', label: `${done.length} ${plural(done.length, 'workflow')} triggered`, detail: names(done), steps };
    }

    return {
        level: 'skipped',
        label: 'No workflow matched',
        detail: skipped.length > 0 ? `${skipped.length} ${plural(skipped.length, 'workflow')} evaluated, none listens to this event` : 'No workflow listens to this event',
        steps
    };
}

/** The first hint whose pattern the message matches, if any. */
export function hintFor(message: string): string {
    if (!message) {
        return undefined;
    }
    return HINTS.find(h => h.pattern.test(message))?.hint;
}

function describeEvent(event: RepositoryHookEvent): string {
    const kind = event.event_type ? `${event.event_name} (${event.event_type})` : `${event.event_name}`;
    const ref = event.extracted_data?.ref;
    const commit = event.extracted_data?.commit;
    return [kind, ref ? `on ${ref}` : null, commit ? `· ${commit.substring(0, 7)}` : null].filter(p => !!p).join(' ');
}

function describeWorkflows(done: Array<RepositoryHookWorkflow>, failed: Array<RepositoryHookWorkflow>, scheduled: Array<RepositoryHookWorkflow>, skipped: Array<RepositoryHookWorkflow>): string {
    const parts: Array<string> = [];
    if (done.length) { parts.push(`triggered: ${names(done)}`); }
    if (failed.length) { parts.push(`failed: ${failed.map(w => w.error ? `${w.workflow_name} (${w.error})` : w.workflow_name).join(', ')}`); }
    if (scheduled.length) { parts.push(`scheduled: ${names(scheduled)}`); }
    if (skipped.length) { parts.push(`skipped: ${names(skipped)}`); }
    return parts.length ? parts.join(' · ') : 'None evaluated';
}

function names(workflows: Array<RepositoryHookWorkflow>): string {
    return workflows.map(w => w.run_number ? `${w.workflow_name} #${w.run_number}` : w.workflow_name).join(', ');
}

function analysisStatus(status: string): StepStatus {
    switch (status) {
        case 'Success': case 'Skipped': return 'finish';
        case 'Error': return 'error';
        case 'InProgress': return 'process';
        default: return 'wait';
    }
}

function firstLine(message: string): string {
    return (message ?? '').split('\n')[0].trim();
}

function plural(n: number, word: string): string {
    return n > 1 ? `${word}s` : word;
}
