import { HookEventWorkflowStatus, RepositoryHookEvent, RepositoryHookWorkflow } from 'app/model/project.model';
import { Initiator } from 'app/model/analysis.model';
import { analysisOutcome } from './analysis-outcome';
import { plural } from './entities';
import { Tone } from './palette';

export interface VerdictStep {
    key: 'received' | 'author' | 'analysis' | 'workflows';
    title: string;
    status: Tone;
    description: string;
}

/** What happened to a repository event, in one line, and step by step. */
export interface HookEventVerdict {
    /** filtered: the workflows ruled the event out themselves · none: nothing listened to it */
    level: Tone;
    label: string;
    detail: string;
    /** Received, author identified, analysis, workflows: always these four, in this order. */
    steps: Array<VerdictStep>;
    /** A way out of the failure, when the message is one the table below knows. */
    hint?: string;
}

// Statuses the hooks service goes through before an event is settled
const IN_PROGRESS = ['Scheduled', 'Analyzing', 'CheckAnalyzing', 'WorkflowHooks', 'GitInfo', 'Workflow'];
// A workflow skipped for one of these reasons did what its definition asks: nothing to fix
const FILTER_SKIP = /no file matches path filters|does not match .*filter|skip CI directive/i;
const ANALYZING = ['Analyzing', 'CheckAnalyzing'];
const TRIGGERING = ['WorkflowHooks', 'GitInfo', 'Workflow'];

const HINTS: Array<{ pattern: RegExp, hint: string }> = [
    { pattern: /User with key/i, hint: 'The commit signing key is not linked to any CDS user: the author must add it to their CDS profile, then push again.' },
    { pattern: /Commit not signed/i, hint: 'Commits must be signed with a key linked to a CDS user for CDS to run anything from them.' },
    { pattern: /unable to get git info/i, hint: 'The ref or commit the event points at could not be read on the repository: check that it still exists and that the vcs credentials of the project can reach it.' },
    { pattern: /no right to trigger (a|this) workflow/i, hint: 'The user the run is attributed to needs the trigger role on this workflow: grant it in the permissions of the project, then push again or start the workflow by hand.' },
    { pattern: /worker model/i, hint: 'The workflow references a worker model that does not exist on this ref: add it under .cds/worker-models or fix its runs-on value.' },
    { pattern: /yaml|unmarshal|unknown field|cannot parse/i, hint: 'A definition file is invalid: fix it on this ref and push again, or trigger an analysis after amending.' }
];

/** The failures of an event the hooks service reports in `last_error`: which step they belong to, and how to name them. */
const EVENT_FAILURES: Array<{ pattern: RegExp, step: number, label: string }> = [
    { pattern: /^User with key/i, step: 1, label: 'Signer unknown' },
    { pattern: /^Commit not signed/i, step: 1, label: 'Commit not signed' },
    { pattern: /unable to get git info/i, step: 3, label: 'Git information unavailable' }
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
    const scheduled = workflows.filter(w => w.status === HookEventWorkflowStatus.Scheduled);
    // A workflow skipped by its own filters behaved as defined; skipped for any other reason, or in
    // error, it did not start because of a problem
    const isFilterSkip = (w: RepositoryHookWorkflow) => w.status === HookEventWorkflowStatus.Skipped && (!w.error || FILTER_SKIP.test(w.error));
    const skipped = workflows.filter(isFilterSkip);
    const notStarted = workflows.filter(w => (w.status === HookEventWorkflowStatus.Error || w.status === HookEventWorkflowStatus.Skipped) && !isFilterSkip(w));
    const analysis = (event.analyses ?? []).find(a => a.project_key === projectKey);
    const inProgress = IN_PROGRESS.indexOf(event.status) !== -1;
    const lastError = cleanErrorMessage(event.last_error);

    const authorName = eventAuthor(event, projectKey);
    const authorKnown = eventAuthorKnown(event, projectKey);
    const received: VerdictStep = { key: 'received', title: 'Event received', status: 'success', description: describeEvent(event) };
    const author: VerdictStep = {
        key: 'author',
        title: 'Author identified',
        status: authorKnown ? 'success' : inProgress ? 'pending' : 'none',
        description: !authorKnown ? 'Not identified'
            : !authorName ? 'Identified'
            : event.sign_key ? `${authorName} · key ${event.sign_key}` : authorName
    };
    const analysisStep: VerdictStep = distant
        ? { key: 'analysis', title: 'Analysis', status: 'none', description: 'Not applicable: the repository is not declared in this project.' }
        : analysis
            ? { key: 'analysis', title: 'Analysis', status: analysisStatus(analysis.status, analysis.error), description: analysis.error ? `${analysis.status} · ${analysis.error}` : analysis.status }
            : { key: 'analysis', title: 'Analysis', status: !inProgress ? 'none' : ANALYZING.indexOf(event.status) !== -1 ? 'processing' : 'pending', description: inProgress ? 'Pending' : 'Not run for this event' };
    // The page lists the workflows itself: the step only speaks when there is none to list, or when
    // something failed at the step itself, above the list
    const workflowsStep: VerdictStep = { key: 'workflows', title: 'Workflows', status: 'none', description: workflows.length ? '' : inProgress ? 'Pending' : 'None evaluated' };
    const steps = [received, author, analysisStep, workflowsStep];

    if (inProgress) {
        workflowsStep.status = TRIGGERING.indexOf(event.status) !== -1 ? 'processing' : 'pending';
        return { level: 'processing', label: 'In progress', detail: event.status, steps };
    }

    if (analysis?.status === 'Error') {
        analysisStep.status = 'error';
        workflowsStep.description = 'Not evaluated: the analysis must succeed first.';
        const analysisError = cleanErrorMessage(analysis.error);
        return { level: 'error', label: 'Analysis failed', detail: firstLine(analysisError) ?? 'Workflows of this ref were not evaluated', steps, hint: hintFor(analysisError || lastError) };
    }

    // The hooks service gives up on an event with a reason, whether it marks it Error or Skipped. A
    // known one names the step that failed; any other only speaks for itself when the workflows do
    // not: when all of them failed to start, it merely repeats their errors.
    const failure = lastError && !analysis ? EVENT_FAILURES.find(f => f.pattern.test(lastError)) : null;
    if (failure) {
        const failed = steps[failure.step];
        failed.status = 'error';
        failed.description = lastError;
        if (failed === author && workflows.length === 0) {
            workflowsStep.description = 'Not evaluated: the author must be identified first.';
        }
        return { level: 'error', label: failure.label, detail: lastError, steps, hint: hintFor(lastError) };
    }

    if (notStarted.length > 0) {
        workflowsStep.status = done.length > 0 ? 'warning' : 'error';
        const errors = [...new Set(notStarted.map(w => cleanErrorMessage(w.error)).filter(e => !!e))].join(' · ');
        const label = done.length > 0
            ? `${notStarted.length} of ${notStarted.length + done.length} did not start`
            : `${notStarted.length} ${plural(notStarted.length, 'workflow')} did not start`;
        return { level: done.length > 0 ? 'warning' : 'error', label, detail: errors, steps, hint: hintFor(errors) };
    }

    if (lastError && !analysis) {
        received.status = 'error';
        received.description = `${received.description} · ${lastError}`;
        return { level: 'error', label: firstLine(lastError) ?? 'Error', detail: lastError, steps, hint: hintFor(lastError) };
    }

    if (scheduled.length > 0) {
        workflowsStep.status = 'processing';
        return { level: 'processing', label: `${scheduled.length} ${plural(scheduled.length, 'workflow')} scheduled`, detail: names(scheduled), steps };
    }

    if (done.length > 0) {
        workflowsStep.status = 'success';
        return { level: 'success', label: `${done.length} ${plural(done.length, 'workflow')} triggered`, detail: names(done), steps };
    }

    const withReason = skipped.filter(w => !!w.error);
    if (withReason.length > 0) {
        workflowsStep.status = 'filtered';
        return {
            level: 'filtered',
            label: `${withReason.length} ${plural(withReason.length, 'workflow')} skipped by ${plural(withReason.length, 'its filters', 'their filters')}`,
            detail: [...new Set(withReason.map(w => w.error))].join(' · '),
            steps
        };
    }
    return {
        level: 'none',
        label: 'No workflow matched',
        detail: skipped.length > 0 ? `${skipped.length} ${plural(skipped.length, 'workflow')} evaluated, none listens to this event` : 'No workflow listens to this event',
        steps
    };
}

/**
 * The message of an error as the API wraps it, without the wrapping: `internal server error (caused
 * by: X (request_id: …))` reads `X`. Nothing to show gives an empty string.
 */
export function cleanErrorMessage(raw: string): string {
    let message = (raw ?? '').trim();
    const wrapped = message.match(/^internal server error \(caused by: (.*)\)$/is);
    if (wrapped) {
        message = wrapped[1];
    }
    return message
        // The hooks service names the workflow it could not run; the page already does
        .replace(/^unable to run workflow \S+:\s*/i, '')
        // `forbidden (from: the cause, request_id: …)`: the cause says it all
        .replace(/\b\w[\w ]*\s\(from: (.*?)(?:, request_id: [^)]*)?\)/gi, '$1')
        .replace(/\s*\(request_id: [^)]*\)/gi, '')
        .replace(/API Error: /gi, '')
        .trim();
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

function names(workflows: Array<RepositoryHookWorkflow>): string {
    return workflows.map(w => w.run_number ? `${w.workflow_name} #${w.run_number}` : w.workflow_name).join(', ');
}

/** The analysis step went through when the analysis ran, even to register nothing. */
function analysisStatus(status: string, error: string): Tone {
    const outcome = analysisOutcome(status, error);
    return outcome === 'none' && status === 'Skipped' ? 'success' : outcome;
}

function firstLine(message: string): string {
    return (message ?? '').split('\n')[0].trim() || null;
}
