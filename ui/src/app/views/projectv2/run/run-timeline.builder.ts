import { TimelineData, TimelineDetail, TimelineLane, TimelineMarker, TimelineSection, TimelineSegment } from "../../../../../libs/timeline/src/public-api";
import { V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatus, WorkflowRunResult, WorkflowRunResultType, WorkflowRunInfo } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";

/** What a lane or a marker of the timeline stands for, so that activating it opens the right panel. */
export interface RunTimelineTarget {
    type: 'job' | 'result' | 'info' | 'hook';
    id: string;
}

export interface RunTimelineBuild {
    data: TimelineData;
    targets: { [key: string]: RunTimelineTarget };
    /** Run job IDs of the chain that set the total duration of the run. */
    criticalPath: Array<string>;
    /** How many jobs the timeline drew, which is what the critical path has to be shorter than to say anything. */
    jobCount: number;
}

/** Lanes of jobs without a stage are gathered here when the workflow also declares stages. */
const NO_STAGE = '__nostage__';

/**
 * A date the API left unset. The engine writes a zero time rather than nothing for some of them,
 * which lands before the epoch once parsed.
 */
function toMs(value: string): number {
    if (!value) {
        return null;
    }
    const ms = Date.parse(value);
    return isNaN(ms) || ms < 0 ? null : ms;
}

/** `os=linux, go=1.22` — what tells two run jobs of the same matrixed job apart. */
function variant(job: V2WorkflowRunJob): string {
    const keys = Object.keys(job.matrix ?? {}).sort();
    return keys.length === 0 ? '' : keys.map(k => `${k}=${job.matrix[k]}`).join(', ');
}

function jobLabel(job: V2WorkflowRunJob): string {
    const v = variant(job);
    return v ? `${job.job_id} (${v})` : job.job_id;
}

/**
 * Turns a workflow run into what the timeline draws.
 *
 * Every job holds one lane, split into the phases it went through: the wait in the queue — which for
 * a job behind a gate can last days — the time a hatchery spent starting a worker, and the run
 * itself. The results of the job are pinned on that lane at the moment they were created, and its
 * steps and results become lanes of their own under it.
 */
export function buildRunTimeline(
    run: V2WorkflowRun,
    jobs: Array<V2WorkflowRunJob>,
    results: Array<WorkflowRunResult>,
    infos: Array<WorkflowRunInfo> = [],
    runAttempt?: number
): RunTimelineBuild {
    const targets: { [key: string]: RunTimelineTarget } = {};
    const criticalPath = computeCriticalPath(run, jobs ?? []);
    const start = timelineStart(run, runAttempt);

    const resultsByJob = new Map<string, Array<WorkflowRunResult>>();
    const orphanResults: Array<WorkflowRunResult> = [];
    (results ?? []).forEach(result => {
        if (toMs(result.issued_at) === null) {
            return;
        }
        const jobID = result.workflow_run_job_id;
        if (!jobID || !(jobs ?? []).some(j => j.id === jobID)) {
            orphanResults.push(result);
            return;
        }
        if (!resultsByJob.has(jobID)) {
            resultsByJob.set(jobID, []);
        }
        resultsByJob.get(jobID).push(result);
    });

    const lanes = (jobs ?? [])
        .map(job => buildJobLane(run, job, resultsByJob.get(job.id) ?? [], criticalPath, targets))
        .filter(entry => !!entry)
        .sort((a, b) => a.at - b.at || a.lane.label.localeCompare(b.lane.label));

    // Stages are drawn in the order they were reached, which is also the order they read in.
    const stages = new Map<string, { label: string, at: number, lanes: Array<TimelineLane> }>();
    lanes.forEach(entry => {
        const stage = entry.stage || NO_STAGE;
        if (!stages.has(stage)) {
            stages.set(stage, { label: stage === NO_STAGE ? null : stage, at: entry.at, lanes: [] });
        }
        stages.get(stage).lanes.push(entry.lane);
    });

    const named = Array.from(stages.entries()).filter(([key]) => key !== NO_STAGE);
    const sections: Array<TimelineSection> = Array.from(stages.entries())
        .sort((a, b) => a[1].at - b[1].at)
        .map(([key, stage]) => <TimelineSection>{
            id: `stage-${key}`,
            // A run without stages needs no heading; one with stages must not leave the rest unlabelled.
            label: stage.label ?? (named.length > 0 ? 'Jobs' : null),
            lanes: stage.lanes
        });

    if (orphanResults.length > 0) {
        sections.push({
            id: 'results',
            label: 'Results',
            lanes: orphanResults.map(result => buildResultLane(result, targets))
        });
    }

    // What set off what is being shown: the hook that started the run where the axis begins with the run,
    // and otherwise the restart that began this attempt, at the head of what it restarted. A mark left at
    // the start of the run on a later attempt would pull the axis back days — see `timelineStart`.
    const trigger = start !== undefined
        ? triggerMarker(run, targets)
        : restartMarker(run, runAttempt ?? run?.run_attempt, lanes.map(entry => entry.at));
    // It goes on a lane of the run itself, in a section of its own above every other: the run belongs to
    // no stage, and putting its lane in the first one said that it was part of it.
    const runLane = buildRunLane(run, infos, targets, trigger);
    if (runLane) {
        sections.unshift({ id: 'run', lanes: [runLane] });
    }

    return {
        data: {
            sections,
            start
        },
        targets,
        criticalPath,
        jobCount: lanes.length
    };
}

/**
 * Where the axis begins.
 *
 * On a first attempt this is the start of the run, so that the crafting that happens before any job is
 * queued is part of what the timeline shows. A restart does not move that date — it only bumps the
 * attempt — so on a later attempt it would sit hours or days before anything of that attempt happened,
 * and the whole timeline would open on a fold. There, the jobs speak for themselves.
 */
function timelineStart(run: V2WorkflowRun, runAttempt?: number): number {
    const attempt = runAttempt ?? run?.run_attempt ?? 1;
    return attempt > 1 ? undefined : (toMs(run?.started) ?? undefined);
}

function buildJobLane(
    run: V2WorkflowRun,
    job: V2WorkflowRunJob,
    jobResults: Array<WorkflowRunResult>,
    criticalPath: Array<string>,
    targets: { [key: string]: RunTimelineTarget }
): { at: number, stage: string, lane: TimelineLane } {
    const queued = toMs(job.queued);
    const scheduled = toMs(job.scheduled);
    const started = toMs(job.started);
    const ended = toMs(job.ended);

    // A job that never reached the queue has nothing to show on a time axis.
    const at = queued ?? scheduled ?? started;
    if (at === null) {
        return null;
    }

    const laneID = `job-${job.id}`;
    targets[laneID] = { type: 'job', id: job.id };

    const segments: Array<TimelineSegment> = [];
    const blocked = job.status === V2WorkflowRunJobStatus.Blocked;
    const waitEnd = scheduled ?? started ?? (isTerminated(job.status) ? ended : null);
    // The phases of a job are one job: grouped, they are drawn as a single thing, so the timeline can
    // neither fold between them nor lose the pointer as it goes from one to the next.
    const group = laneID;
    if (queued !== null) {
        segments.push({
            id: `${laneID}-queued`,
            group,
            start: queued,
            end: waitEnd,
            // A job held by a gate or a concurrency rule waits in the queue like any other, but for a
            // reason worth naming: that wait is the one that can last days.
            kind: blocked ? 'blocked' : 'queued',
            label: blocked ? 'Blocked' : 'Queued',
            // Nothing is being spent on the job while it waits, so the axis is free to fold that
            // wait — which is the only way a gate held for days can be drawn alongside the rest.
            idle: true
        });
    }
    if (scheduled !== null) {
        segments.push({
            id: `${laneID}-scheduling`,
            group,
            start: scheduled,
            end: started ?? (isTerminated(job.status) ? ended : null),
            kind: 'scheduling',
            label: 'Scheduling'
        });
    }
    if (started !== null) {
        segments.push({
            id: `${laneID}-running`,
            group,
            start: started,
            end: ended,
            kind: `running-${(job.status ?? '').toLowerCase()}`,
            label: 'Running'
        });
    }

    const markers: Array<TimelineMarker> = [];
    const gate = gateMarker(run, job, at);
    if (gate) {
        markers.push(gate);
    }
    markers.push(...jobResults.map(result => {
        const markerID = `result-${result.id}`;
        targets[markerID] = { type: 'result', id: result.id };
        return resultMarker(markerID, result);
    }));

    /**
     * A result says which job made it, never which step. But it is created by the step that is running at
     * the time, and both timestamps come off the same worker, so the step whose run covers the moment the
     * result was created is the step that created it. Results whose moment matches no step — there should
     * be none, but a result created between two steps would be one — keep a lane of their own below.
     */
    const stepOf = (result: WorkflowRunResult): string => {
        const at = toMs(result.issued_at);
        const match = Object.entries(job.steps_status ?? {}).find(([, status]) => {
            const from = toMs(status.started);
            const to = toMs(status.ended);
            return from !== null && at >= from && (to === null || at <= to);
        });
        return match ? match[0] : null;
    };
    const resultsByStep = new Map<string, Array<WorkflowRunResult>>();
    const looseResults: Array<WorkflowRunResult> = [];
    jobResults.forEach(result => {
        const step = stepOf(result);
        if (!step) {
            looseResults.push(result);
            return;
        }
        if (!resultsByStep.has(step)) {
            resultsByStep.set(step, []);
        }
        resultsByStep.get(step).push(result);
    });

    const steps: Array<TimelineLane> = Object.entries(job.steps_status ?? {})
        .map(([name, status]) => ({ name, status, at: toMs(status.started) }))
        .filter(step => step.at !== null)
        .sort((a, b) => a.at - b.at)
        .map(step => <TimelineLane>{
            id: `${laneID}-step-${step.name}`,
            label: step.name,
            status: (step.status.conclusion ?? '').toLowerCase(),
            segments: [{
                id: `${laneID}-step-${step.name}-run`,
                start: step.at,
                end: toMs(step.status.ended),
                kind: `step-${(step.status.conclusion ?? 'running').toLowerCase()}`,
                label: step.name
            }],
            // Pinned on the step that made them, so unfolding a job says which of its steps produced what.
            markers: (resultsByStep.get(step.name) ?? []).map(result => {
                const markerID = `result-${result.id}-step`;
                targets[markerID] = { type: 'result', id: result.id };
                return resultMarker(markerID, result);
            })
        });

    // The dates the graph used to show on hovering a node. Only the dates: the status is in the colour
    // of the bars, and how long each phase took is already the breakdown above them.
    const details: Array<TimelineDetail> = [
        { label: 'Queued', at: queued },
        { label: 'Scheduled', at: scheduled },
        { label: 'Started', at: started },
        { label: 'Ended', at: ended }
    ].filter(entry => entry.at !== null)
        .map(entry => ({ label: `${entry.label}:`, value: new Date(entry.at).toLocaleString() }));
    if (job.retry > 0) {
        details.push({ label: 'Retry:', value: `${job.retry}` });
    }

    return {
        at,
        stage: job.job?.stage,
        lane: {
            id: laneID,
            label: jobLabel(job),
            // No second line under the name: the status is in the colour of the bars, and a bare
            // duration there raised more questions than it answered. Both are in the hover card, where
            // there is room to say what they are.
            details,
            status: (job.status ?? '').toLowerCase(),
            segments,
            markers,
            // Results sit on the step that made them. Only the ones no step accounts for get a lane, so
            // that nothing is ever dropped for want of a step to hang it on.
            lanes: steps.concat(looseResults.map(result => buildResultLane(result, targets))),
            highlighted: criticalPath.indexOf(job.id) !== -1,
            activatable: true
        }
    };
}

/**
 * The lane of the run itself: what set it off, and what it logged as an error. Neither is about any one
 * job — a run info carries no job of its own — and both are instants, so the lane has markers and no bars.
 *
 * Warnings and plain infos are left out on purpose: they are the ordinary chatter of a run, and a mark for
 * each of them would bury the one kind worth stopping at. The Info tab still has all of them.
 */
function buildRunLane(
    run: V2WorkflowRun,
    infos: Array<WorkflowRunInfo>,
    targets: { [key: string]: RunTimelineTarget },
    trigger: TimelineMarker
): TimelineLane {
    const errors = (infos ?? [])
        .filter(info => info.level === 'error' && toMs(info.issued_at) !== null)
        .sort((a, b) => toMs(a.issued_at) - toMs(b.issued_at));

    const markers: Array<TimelineMarker> = [];
    if (trigger) {
        markers.push(trigger);
    }
    errors.forEach(info => {
        const markerID = `info-${info.id}`;
        targets[markerID] = { type: 'info', id: info.id };
        markers.push(<TimelineMarker>{
            id: markerID,
            at: toMs(info.issued_at),
            kind: 'error',
            icon: 'close-circle',
            // The message is the whole of it, so it is the heading rather than a line of detail.
            label: info.message,
            plural: 'errors',
            details: [{ label: 'Logged:', value: new Date(toMs(info.issued_at)).toLocaleString() }]
        });
    });
    if (markers.length === 0) {
        return null;
    }

    return {
        id: 'run',
        // Not the name of the workflow, which reads like the name of a job. This lane is the run.
        label: 'Run',
        sublabel: errors.length > 0 ? `${errors.length} error${errors.length > 1 ? 's' : ''}` : null,
        status: errors.length > 0 ? 'error' : 'run',
        segments: [],
        markers
    };
}

/** What set the run off, at the moment it started. */
function triggerMarker(run: V2WorkflowRun, targets: { [key: string]: RunTimelineTarget }): TimelineMarker {
    const at = toMs(run?.started);
    if (at === null) {
        return null;
    }

    const event = run?.event;
    const name = event?.event_name ?? '';
    const markerID = 'run-trigger';
    // The hook panel is opened the way the graph opens it, by the name of the hook that fired.
    targets[markerID] = { type: 'hook', id: name };

    const details: Array<TimelineDetail> = [];
    if (event?.hook_type) {
        details.push({ label: 'Hook:', value: hookLabel(event.hook_type) });
    }
    if (run?.username) {
        details.push({ label: 'By:', value: run.username });
    }
    if (event?.ref) {
        details.push({ label: 'Ref:', value: shortRef(event.ref) });
    }
    if (event?.sha) {
        details.push({ label: 'Commit:', value: event.sha.substring(0, 8) });
    }
    if (event?.entity_updated) {
        details.push({ label: 'Updated:', value: event.entity_updated });
    }
    details.push({ label: 'Started:', value: new Date(at).toLocaleString() });

    return {
        id: markerID,
        at,
        kind: 'trigger',
        icon: triggerIcon(name),
        label: triggerLabel(name),
        details
    };
}

/**
 * What set a later attempt off: someone asking for jobs to be run again. It sits at the head of what was
 * restarted, which is where the axis of that attempt begins.
 *
 * Nothing opens from it — there is no panel for a restart — so like a gate it is a mark to hover. Who
 * asked, and for what, is on the run: the API records one event per job asked for, stamped with the
 * attempt it created.
 */
function restartMarker(run: V2WorkflowRun, attempt: number, laneStarts: Array<number>): TimelineMarker {
    const at = Math.min(...laneStarts.filter(value => value !== null && isFinite(value)));
    if (!isFinite(at)) {
        return null;
    }

    const asked = (run?.job_events ?? []).filter(event => event.run_attempt === attempt);
    const by = asked.map(event => event.username).filter(name => !!name);
    const jobs = asked.map(event => event.job_id).filter(id => !!id);

    const details: Array<TimelineDetail> = [];
    if (jobs.length > 0) {
        details.push({ label: jobs.length > 1 ? 'Jobs:' : 'Job:', value: [...new Set(jobs)].join(', ') });
    }
    details.push({ label: 'Attempt:', value: `${attempt}` });
    details.push({ label: 'Started:', value: new Date(at).toLocaleString() });

    return {
        id: 'run-restart',
        at,
        kind: 'trigger',
        icon: 'redo',
        label: by.length > 0 ? `Restarted by ${by[0]}` : 'Restarted',
        details
    };
}

/** How the run came to be, as it reads to someone who did not start it. */
function triggerLabel(eventName: string): string {
    switch (eventName) {
        case 'manual': return 'Started by hand';
        case 'push': return 'Started by a push';
        case 'pull-request': return 'Started by a pull request';
        case 'pull-request-comment': return 'Started by a comment on a pull request';
        case 'webhook': return 'Started by a web hook';
        case 'scheduler': return 'Started by the scheduler';
        case 'workflow-run': return 'Started by another run';
        case 'workflow-update': return 'Started by a change to the workflow';
        case 'model-update': return 'Started by a change to a worker model';
        default: return eventName ? `Started by ${eventName}` : 'Run started';
    }
}

/** The kind of hook behind the event, as the API names it. */
function hookLabel(hookType: string): string {
    switch (hookType) {
        case 'RepositoryWebHook': return 'Repository web hook';
        case 'WorkerModelUpdate': return 'Worker model update';
        case 'WorkflowUpdate': return 'Workflow update';
        case 'WorkflowRun': return 'Workflow run';
        case 'Webhook': return 'Web hook';
        default: return hookType;
    }
}

function triggerIcon(eventName: string): string {
    switch (eventName) {
        case 'manual': return 'user';
        case 'push':
        case 'pull-request':
        case 'pull-request-comment':
            return 'branches';
        case 'scheduler': return 'clock-circle';
        case 'webhook': return 'global';
        case 'workflow-run': return 'appstore';
        case 'workflow-update':
        case 'model-update':
            return 'sync';
        default: return 'thunderbolt';
    }
}

function shortRef(ref: string): string {
    return ref.replace(/^refs\/(heads|tags)\//, '');
}

function buildResultLane(result: WorkflowRunResult, targets: { [key: string]: RunTimelineTarget }): TimelineLane {
    const laneID = `result-lane-${result.id}`;
    targets[laneID] = { type: 'result', id: result.id };
    return {
        id: laneID,
        label: resultLabel(result),
        sublabel: result.type,
        details: resultDetails(result),
        status: 'result',
        segments: [],
        markers: [resultMarker(`${laneID}-marker`, result)],
        activatable: true
    };
}

/**
 * What a result looks like on a lane: an icon saying what kind of thing it is, and enough about it to
 * tell it from the others without opening it.
 */
function resultMarker(id: string, result: WorkflowRunResult): TimelineMarker {
    return {
        id,
        at: toMs(result.issued_at),
        kind: `result-${result.type}`,
        icon: resultIcon(result.type),
        label: resultLabel(result),
        details: resultDetails(result)
    };
}

function resultDetails(result: WorkflowRunResult): Array<TimelineDetail> {
    // No type: the key already begins with it.
    const details: Array<TimelineDetail> = [
        { label: 'Created:', value: new Date(toMs(result.issued_at)).toLocaleString() }
    ];
    const name = result.detail?.data?.name;
    if (name && name !== result.identifier) {
        details.push({ label: 'File:', value: `${name}` });
    }
    const size = result.detail?.data?.size;
    if (typeof size === 'number' && size > 0) {
        details.push({ label: 'Size:', value: formatBytes(size) });
    }
    return details;
}

/**
 * The icon standing for a kind of result. A test report and a Docker image are not the same thing and
 * should not be the same lozenge; anything unrecognised is a file, which every result ultimately is.
 */
function resultIcon(type: WorkflowRunResultType | string): string {
    switch (type) {
        case 'tests': return 'experiment';
        case 'coverage': return 'pie-chart';
        case 'release': return 'tag';
        case 'variable': return 'code';
        case 'docker': return 'container';
        case 'deployment': return 'rocket';
        case 'staticFiles': return 'global';
        case 'helm': return 'build';
        case 'debian':
        case 'python':
        case 'npm':
        case 'maven':
        case 'gradle':
        case 'sbt':
        case 'nuget':
        case 'conan':
        case 'puppet':
        case 'terraformModule':
        case 'terraformProvider':
            return 'inbox';
        default: return 'file';
    }
}

function formatBytes(size: number): string {
    const units = ['B', 'kB', 'MB', 'GB', 'TB'];
    let value = size;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024;
        unit++;
    }
    return `${unit === 0 ? value : value.toFixed(1)} ${units[unit]}`;
}

/**
 * What a result is called here. `identifier` is what the API calls its name — `generic:binary.tar.gz`,
 * `tests:unit.xml` — which is what the Results tab lists it under and what someone looking for it in a
 * log would search for. `label` is a sentence about it instead, and its size and filename are already in
 * the detail, so the key is what belongs on the timeline.
 */
function resultLabel(result: WorkflowRunResult): string {
    return result.identifier || result.label || result.type;
}

function isTerminated(status: V2WorkflowRunJobStatus): boolean {
    switch (status) {
        case V2WorkflowRunJobStatus.Cancelled:
        case V2WorkflowRunJobStatus.Fail:
        case V2WorkflowRunJobStatus.Stopped:
        case V2WorkflowRunJobStatus.Success:
        case V2WorkflowRunJobStatus.Skipped:
            return true;
    }
    return false;
}

/**
 * The chain of jobs that set the total duration of the run: the last one to end, then whichever of
 * the jobs it waited on ended last, and so on back to the start. Shortening any other job would not
 * have made the run shorter.
 */
export function computeCriticalPath(run: V2WorkflowRun, jobs: Array<V2WorkflowRunJob>): Array<string> {
    const workflow = run?.workflow_data?.workflow;
    // Every job that ran and finished. A skipped job is terminated and carries an end date like any
    // other — the engine stamps one on whatever it finishes with — but it did no work, and a chain of
    // the jobs that set the duration of the run has no business claiming one of those set anything.
    const worked = (jobs ?? []).filter(job => toMs(job.ended) !== null && didWork(job.status));
    if (!workflow || worked.length === 0) {
        return [];
    }

    const endOf = (job: V2WorkflowRunJob) => toMs(job.ended);
    const latest = (candidates: Array<V2WorkflowRunJob>) =>
        candidates.reduce((best, job) => endOf(job) > endOf(best) ? job : best, candidates[0]);

    /**
     * The jobs a job waited on that actually ran. A job it waited on which was skipped did not hold it
     * up by working, so the chain carries on through whatever *that* one waited on instead, rather than
     * stopping at it or crediting it.
     */
    const workingPredecessors = (jobID: string, guard: Set<string>): Array<V2WorkflowRunJob> => {
        const found: Array<V2WorkflowRunJob> = [];
        needsOf(workflow, jobID).forEach(name => {
            if (guard.has(name)) {
                return;
            }
            guard.add(name);
            const ran = worked.filter(job => job.job_id === name);
            if (ran.length > 0) {
                found.push(...ran);
                return;
            }
            found.push(...workingPredecessors(name, guard));
        });
        return found;
    };

    const path: Array<string> = [];
    const seen = new Set<string>();
    let current = latest(worked);

    while (current && !seen.has(current.id)) {
        seen.add(current.id);
        path.push(current.id);
        const predecessors = workingPredecessors(current.job_id, new Set<string>([current.job_id]))
            .filter(job => !seen.has(job.id));
        current = predecessors.length > 0 ? latest(predecessors) : null;
    }

    return path;
}

/** Whether a job spent time doing something, as opposed to being passed over. */
function didWork(status: V2WorkflowRunJobStatus): boolean {
    return status !== V2WorkflowRunJobStatus.Skipped;
}

/**
 * The mark at the head of a gated job's lane: what held it, and what was answered to let it through.
 *
 * The definition of the gate lives on the workflow, the values it was triggered with on the run job. A
 * job still waiting has none of the latter, and says so rather than showing an empty list.
 */
function gateMarker(run: V2WorkflowRun, job: V2WorkflowRunJob, at: number): TimelineMarker {
    const name = job.job?.gate;
    if (!name) {
        return null;
    }
    const definition = run?.workflow_data?.workflow?.gates?.[name];
    const answered = job.gate_inputs ?? {};
    const details: Array<TimelineDetail> = Object.keys(answered).sort()
        .map(key => ({ label: `${key}:`, value: formatGateInput(answered[key]) }));

    if (details.length === 0) {
        details.push({
            label: job.status === V2WorkflowRunJobStatus.Blocked ? 'Waiting to be triggered' : 'No inputs',
            value: ''
        });
    }
    if (definition?.if) {
        details.push({ label: 'Condition:', value: definition.if });
    }

    return {
        id: `gate-${job.id}`,
        at,
        kind: 'gate',
        icon: 'unlock',
        label: `Gate ${name}`,
        details
    };
}

function formatGateInput(value: any): string {
    if (value === null || value === undefined || value === '') {
        return '—';
    }
    return typeof value === 'object' ? JSON.stringify(value) : `${value}`;
}

/** The jobs a job waits on: the ones it needs, plus every job of the stages its stage needs. */
function needsOf(workflow: any, jobID: string): Array<string> {
    const definition = workflow?.jobs?.[jobID];
    if (!definition) {
        return [];
    }
    const needs: Array<string> = [...(definition.needs ?? [])];
    const stageNeeds: Array<string> = definition.stage ? (workflow.stages?.[definition.stage]?.needs ?? []) : [];
    if (stageNeeds.length > 0) {
        Object.keys(workflow.jobs ?? {}).forEach(other => {
            if (stageNeeds.indexOf(workflow.jobs[other]?.stage) !== -1) {
                needs.push(other);
            }
        });
    }
    return needs;
}
