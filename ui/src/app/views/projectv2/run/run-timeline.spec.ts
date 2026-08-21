import { axisEnds, AXIS_BAND_PX, formatDuration, MARKER_ROOM_PX, revealBy, shouldFollow, timelineIcon, TimelineLane, TimelineScale, TimelineSection } from "../../../../../libs/timeline/src/public-api";
import { V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatus, WorkflowRunInfo, WorkflowRunResult } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { buildRunTimeline, computeCriticalPath } from "./run-timeline.builder";

const MINUTE = 60000;
const HOUR = 3600000;

/** Two minutes of work two hours apart, which is the shape a gate leaves behind. */
const BUSY_ACROSS_A_GATE = [
    { start: 0, end: MINUTE },
    { start: 2 * HOUR + MINUTE, end: 2 * HOUR + 2 * MINUTE }
];
const SPAN_ACROSS_A_GATE = 2 * HOUR + 2 * MINUTE;

describe('TimelineScale', () => {
    it('leaves time proportional when it is asked not to fold', () => {
        const scale = TimelineScale.build(0, SPAN_ACROSS_A_GATE, BUSY_ACROSS_A_GATE, { foldGapsLongerThanMs: 0, foldedGapShare: 0.03 });

        expect(scale.folds.length).toBe(0);
        // The whole of the first job holds less than one percent of the axis.
        expect(scale.ratio(MINUTE)).toBeCloseTo(MINUTE / SPAN_ACROSS_A_GATE, 5);
    });

    it('folds an idle stretch so that what ran keeps the axis', () => {
        const scale = TimelineScale.build(0, SPAN_ACROSS_A_GATE, BUSY_ACROSS_A_GATE, { foldGapsLongerThanMs: MINUTE, foldedGapShare: 0.03 });

        expect(scale.folds.length).toBe(1);
        expect(TimelineScale.foldLabel(scale.folds[0])).toBe('2h');

        // Two minutes of work and a two hour wait: the work now holds almost all of the axis, and the
        // fold the little that is left.
        expect(scale.ratio(MINUTE)).toBeGreaterThan(0.45);
        expect(scale.ratio(2 * HOUR + MINUTE)).toBeLessThan(0.55);
        expect(scale.ratio(2 * HOUR + MINUTE) - scale.ratio(MINUTE)).toBeCloseTo(0.03 / 1.03, 3);
    });

    it('keeps an instant inside an idle stretch out of the fold', () => {
        const scale = TimelineScale.build(0, SPAN_ACROSS_A_GATE,
            BUSY_ACROSS_A_GATE.concat([{ start: HOUR, end: HOUR }]),
            { foldGapsLongerThanMs: MINUTE, foldedGapShare: 0.03 });

        // The wait is now two folds with the instant between them, instead of one fold hiding it.
        expect(scale.folds.length).toBe(2);
        expect(scale.ratio(HOUR)).toBeGreaterThan(scale.ratio(MINUTE));
        expect(scale.ratio(HOUR)).toBeLessThan(scale.ratio(2 * HOUR + MINUTE));
    });

    it('reads a position back as the date it stands for', () => {
        const scale = TimelineScale.build(0, SPAN_ACROSS_A_GATE, BUSY_ACROSS_A_GATE, { foldGapsLongerThanMs: MINUTE, foldedGapShare: 0.03 });

        [0, 30000, MINUTE, HOUR, 2 * HOUR + 90000, SPAN_ACROSS_A_GATE].forEach(at => {
            expect(scale.time(scale.ratio(at))).toBeCloseTo(at, -1);
        });
    });

    it('graduates only what is drawn to scale', () => {
        const scale = TimelineScale.build(0, SPAN_ACROSS_A_GATE, BUSY_ACROSS_A_GATE, { foldGapsLongerThanMs: MINUTE, foldedGapShare: 0.03 });

        const fold = scale.folds[0];
        scale.ticks(0, 1, 10).forEach(tick => {
            expect(tick.at >= fold.start && tick.at <= fold.end && tick.at !== fold.start && tick.at !== fold.end)
                .withContext(`tick at ${tick.at} fell inside the fold`).toBeFalse();
        });
    });

    it('places everything at the start when there is no length to speak of', () => {
        const scale = TimelineScale.build(1000, 1000, [{ start: 1000, end: 1000 }]);

        expect(scale.ratio(1000)).toBe(0);
        expect(scale.ratio(2000)).toBe(1);
    });
});

/**
 * Room for the ends of an axis is taken out of the *track*, in pixels, and never out of the stretch of
 * time it covers. Time taken has to be either drawn to scale — where a few pixels of room on a two day
 * axis is an hour of invented time that then takes the whole view — or folded away, which is the room
 * being taken straight back.
 */
describe('axisEnds', () => {
    const ends = { trackWidth: 800, lasting: true, markAtStart: false, markAtEnd: false };

    it('keeps the two bands clear and nothing more', () => {
        expect(axisEnds(ends)).toEqual({ start: AXIS_BAND_PX, end: AXIS_BAND_PX });
    });

    it('makes room on the side a marker sits on the bound of', () => {
        // A bar reaching the edge of the axis still reads as a bar; a glyph centred on that edge is half
        // drawn outside it. So that side keeps more clear, and only that side.
        expect(axisEnds({ ...ends, markAtStart: true }))
            .toEqual({ start: AXIS_BAND_PX + MARKER_ROOM_PX, end: AXIS_BAND_PX });
    });

    it('gives content that never lasted the middle third', () => {
        // A run that failed before queueing a job is two instants and not one bar: its bounds *are* the
        // instants, so drawn across the whole track they sit against its two edges with nothing between.
        const middle = axisEnds({ ...ends, lasting: false });

        expect(middle.start).toBe(middle.end);
        // A third of what the bands leave, which is the axis being drawn in the middle of what is left.
        expect(800 - middle.start - middle.end).toBeCloseTo((800 - 2 * AXIS_BAND_PX) / 3, 6);
    });

    it('leaves the plot a fifth of the track however narrow it gets', () => {
        const tight = axisEnds({ ...ends, trackWidth: 50, lasting: false, markAtStart: true, markAtEnd: true });

        expect(tight.start + tight.end).toBeCloseTo(40, 6);
    });

});

describe('timelineIcon', () => {
    it('draws only the icons the view carries', () => {
        expect(timelineIcon('experiment')).toBe('experiment');
    });

    it('draws no icon at all rather than one it was told to go and find', () => {
        // An unregistered name is not a missing icon: the icon component fetches `assets/<theme>/<name>.svg`
        // and puts what comes back on the page. A name reaching the view from data — a result type, a file
        // name — would be a way of choosing a URL to load and render.
        ['nonsense', '../../../etc/passwd', 'https://elsewhere.example/x', '', null].forEach(name => {
            expect(timelineIcon(name)).toBeNull();
        });
    });

    it('never lets one of the run through the adapter either', () => {
        const run = <V2WorkflowRun>{ started: '2026-01-01T10:00:00Z', workflow_data: { workflow: <any>{ jobs: {} } } };
        const nasty = <WorkflowRunResult>{
            id: 'r1', issued_at: '2026-01-01T10:00:01Z',
            type: <any>'../../../evil', identifier: 'x', label: 'x'
        };
        const lane = buildRunTimeline(run, [], [nasty]).data.sections.flatMap(jobLanes)[0];

        expect(lane.markers[0].icon).toBe('file');
    });
});

describe('formatDuration', () => {
    it('keeps the largest unit and the one below it', () => {
        expect(formatDuration(0)).toBe('<1s');
        expect(formatDuration(999)).toBe('<1s');
        expect(formatDuration(30000)).toBe('30s');
        expect(formatDuration(61000)).toBe('1m 1s');
        expect(formatDuration(90000)).toBe('1m 30s');
        expect(formatDuration(2 * HOUR)).toBe('2h');
        expect(formatDuration(HOUR + MINUTE + 1000)).toBe('1h 1m');
        expect(formatDuration(86400000)).toBe('1d');
        expect(formatDuration(2 * 86400000 + 3 * HOUR)).toBe('2d 3h');
    });

    it('drops what says nothing at the scale it is read at', () => {
        // Two days and two seconds is two days; the seconds are noise next to the days.
        expect(formatDuration(2 * 86400000 + 2000)).toBe('2d');
    });
});

/**
 * The run holds a section of its own above the jobs — what set it off, and anything it logged as an
 * error — so the lanes of the jobs are those of every other section.
 */
const jobLanes = (section: TimelineSection): Array<TimelineLane> => section.id === 'run' ? [] : section.lanes;
const jobSections = (built: ReturnType<typeof buildRunTimeline>): Array<TimelineSection> =>
    built.data.sections.filter(section => section.id !== 'run');
const runLane = (built: ReturnType<typeof buildRunTimeline>): TimelineLane =>
    built.data.sections.find(section => section.id === 'run')?.lanes[0];

function job(overrides: Partial<V2WorkflowRunJob>): V2WorkflowRunJob {
    return <V2WorkflowRunJob>{
        status: V2WorkflowRunJobStatus.Success,
        steps_status: {},
        ...overrides
    };
}

describe('computeCriticalPath', () => {
    const run = <V2WorkflowRun>{
        workflow_data: {
            workflow: <any>{
                jobs: {
                    a: {},
                    b: { needs: ['a'] },
                    c: { needs: ['a'] },
                    d: { needs: ['b', 'c'] }
                }
            }
        }
    };

    it('follows the jobs that set the total duration of the run', () => {
        const jobs = [
            job({ id: 'ja', job_id: 'a', ended: '2026-01-01T10:00:10Z' }),
            job({ id: 'jb', job_id: 'b', ended: '2026-01-01T10:00:50Z' }),
            job({ id: 'jc', job_id: 'c', ended: '2026-01-01T10:00:30Z' }),
            job({ id: 'jd', job_id: 'd', ended: '2026-01-01T10:01:00Z' })
        ];

        // c ended before b, so shortening c would not have made the run any shorter.
        expect(computeCriticalPath(run, jobs)).toEqual(['jd', 'jb', 'ja']);
    });

    it('has nothing to follow while no job has ended', () => {
        expect(computeCriticalPath(run, [job({ id: 'ja', job_id: 'a' })])).toEqual([]);
    });

    it('says how many jobs it drew, so a chain of all of them can be told apart', () => {
        // The highlight works by holding back what is not on the chain, so a chain holding every job of
        // the run — which is what one job after another comes to — has nothing to bring out.
        const chained = <V2WorkflowRun>{
            workflow_data: { workflow: <any>{ jobs: { a: {}, b: { needs: ['a'] }, c: { needs: ['b'] } } } }
        };
        const at = (s: string) => ({ queued: s, started: s, ended: s });
        const jobs = [
            job({ id: 'ja', job_id: 'a', ...at('2026-01-01T10:00:10Z') }),
            job({ id: 'jb', job_id: 'b', ...at('2026-01-01T10:00:20Z') }),
            job({ id: 'jc', job_id: 'c', ...at('2026-01-01T10:00:30Z') })
        ];
        const built = buildRunTimeline(chained, jobs, []);

        expect(built.criticalPath.length).toBe(3);
        expect(built.jobCount).toBe(3);
    });
});

describe('buildRunTimeline', () => {
    const run = <V2WorkflowRun>{
        started: '2026-01-01T10:00:00Z',
        workflow_data: { workflow: <any>{ jobs: { build: {} } } }
    };

    const runJob = job({
        id: 'j1',
        job_id: 'build',
        queued: '2026-01-01T10:00:01Z',
        scheduled: '2026-01-01T10:00:05Z',
        started: '2026-01-01T10:00:20Z',
        ended: '2026-01-01T10:01:00Z',
        steps_status: {
            checkout: <any>{ conclusion: 'Success', started: '2026-01-01T10:00:21Z', ended: '2026-01-01T10:00:30Z' }
        }
    });

    const result = <WorkflowRunResult>{
        id: 'r1',
        workflow_run_job_id: 'j1',
        issued_at: '2026-01-01T10:00:55Z',
        type: 'generic',
        label: 'binary.tar.gz'
    };

    it('splits a job into the phases it went through', () => {
        const built = buildRunTimeline(run, [runJob], []);
        const lane = built.data.sections.flatMap(jobLanes)[0];

        expect(lane.segments.map(s => s.kind)).toEqual(['queued', 'scheduling', 'running-success']);
        expect(lane.segments[0].end - lane.segments[0].start).toBe(4000);
        expect(lane.segments[1].end - lane.segments[1].start).toBe(15000);
        expect(lane.segments[2].end - lane.segments[2].start).toBe(40000);
        // Nothing under the name: a bare duration there said neither what it measured nor of what.
        expect(lane.sublabel).toBeUndefined();
        // The axis starts with the run, not with its first job.
        expect(built.data.start).toBe(Date.parse(run.started));
    });

    it('makes the phases of a job one group', () => {
        const lane = buildRunTimeline(run, [runJob], []).data.sections.flatMap(jobLanes)[0];

        // One group, so the bars are drawn as one element: the pointer cannot fall between two phases,
        // and no fold can open between them.
        expect([...new Set(lane.segments.map(s => s.group))]).toEqual([lane.id]);
    });

    it('leaves no gap between the phases of a job', () => {
        const lane = buildRunTimeline(run, [runJob], []).data.sections.flatMap(jobLanes)[0];

        // Why holding a group together costs nothing here: a job leaving one status takes the next in
        // the same instant, so there is never a gap inside the group to hold open.
        const ordered = lane.segments.slice().sort((a, b) => a.start - b.start);
        ordered.slice(1).forEach((segment, i) => expect(segment.start).toBe(ordered[i].end));
    });

    it('names the wait of a job held by a gate', () => {
        const blocked = job({ id: 'j2', job_id: 'build', status: V2WorkflowRunJobStatus.Blocked, queued: '2026-01-01T10:00:01Z' });
        const lane = buildRunTimeline(run, [blocked], []).data.sections.flatMap(jobLanes)[0];

        expect(lane.segments.length).toBe(1);
        expect(lane.segments[0].kind).toBe('blocked');
        // Still waiting: the segment is left open so that it grows on screen.
        expect(lane.segments[0].end).toBeNull();
    });

    it('marks waiting as idle, so a wait of days can be folded', () => {
        const built = buildRunTimeline(run, [runJob], []);
        const lane = built.data.sections.flatMap(jobLanes)[0];

        // Queueing and being blocked are waiting; scheduling a worker and running are not.
        expect(lane.segments.filter(s => s.idle).map(s => s.kind)).toEqual(['queued']);
        expect(lane.segments.filter(s => !s.idle).map(s => s.kind)).toEqual(['scheduling', 'running-success']);

        const blocked = job({ id: 'j2', job_id: 'build', status: V2WorkflowRunJobStatus.Blocked, queued: '2026-01-01T10:00:01Z' });
        expect(buildRunTimeline(run, [blocked], []).data.sections.flatMap(jobLanes)[0].segments[0].idle).toBeTrue();
    });

    it('folds through a blocked job, whose phases are a group of one', () => {
        // The rule holding a group together claims what sits *between* its segments, never what is inside
        // one of them. A job held by a gate is a single segment spanning the whole wait, and folding
        // through it is the point of the whole thing.
        const gated = job({ id: 'j2', job_id: 'build', status: V2WorkflowRunJobStatus.Blocked, queued: '2026-01-01T10:00:00Z' });
        const lane = buildRunTimeline(run, [gated], []).data.sections.flatMap(jobLanes)[0];

        expect(lane.segments.length).toBe(1);
        const wait = 2 * 24 * HOUR;
        const scale = TimelineScale.build(lane.segments[0].start, lane.segments[0].start + wait, []);
        expect(scale.folds.length).toBe(1);
    });

    it('folds the wait of a gate without folding the jobs around it', () => {
        // One job before the gate, one after it two days later, and the gated job waiting in between.
        const before = job({
            id: 'j1', job_id: 'build',
            queued: '2026-01-01T10:00:00Z', scheduled: '2026-01-01T10:00:05Z',
            started: '2026-01-01T10:00:10Z', ended: '2026-01-01T10:01:00Z'
        });
        const gated = job({
            id: 'j2', job_id: 'deploy', status: V2WorkflowRunJobStatus.Blocked,
            queued: '2026-01-01T10:01:00Z'
        });
        const after = job({
            id: 'j3', job_id: 'notify',
            queued: '2026-01-03T10:01:00Z', scheduled: '2026-01-03T10:01:02Z',
            started: '2026-01-03T10:01:05Z', ended: '2026-01-03T10:02:00Z'
        });

        const built = buildRunTimeline(run, [before, gated, after], []);
        const busy = built.data.sections
            .reduce((all, section) => all.concat(section.lanes), [])
            .reduce((all, lane) => all.concat(lane.segments.filter(s => !s.idle)), [])
            .map(s => ({ start: s.start, end: s.end }));

        const from = Date.parse('2026-01-01T10:00:00Z');
        const to = Date.parse('2026-01-03T10:02:00Z');
        const scale = TimelineScale.build(from, to, busy);

        // Two days of waiting, two minutes of work: the work must still own most of the axis.
        expect(scale.folds.length).toBe(1);
        expect(TimelineScale.foldLabel(scale.folds[0])).toBe('2d');
        expect(scale.ratio(Date.parse('2026-01-01T10:01:00Z'))).toBeGreaterThan(0.4);
    });

    it('gives each kind of result the icon that says what it is', () => {
        const of = (type: string) => <WorkflowRunResult>{
            id: `r-${type}`, workflow_run_job_id: 'j1', issued_at: '2026-01-01T10:00:55Z', type, label: type
        };
        const kinds = ['generic', 'tests', 'coverage', 'docker', 'npm', 'somethingNobodyHasSeenYet'];
        const lane = buildRunTimeline(run, [runJob], kinds.map(of)).data.sections.flatMap(jobLanes)[0];

        // The shape tells two results apart, so the colour no longer has to.
        expect(lane.markers.map(m => m.icon)).toEqual(['file', 'experiment', 'pie-chart', 'container', 'inbox', 'file']);
    });

    it('calls a result by its key rather than by a sentence about it', () => {
        const sized = <WorkflowRunResult>{
            ...result,
            identifier: 'generic:binary.tar.gz',
            label: 'Filename: binary.tar.gz - Size: 2.0 kB',
            detail: <any>{ data: { name: 'binary.tar.gz', size: 2048 } }
        };
        const lane = buildRunTimeline(run, [runJob], [sized]).data.sections.flatMap(jobLanes)[0];

        // The key is what the Results tab lists it under and what someone would search a log for.
        expect(lane.markers[0].label).toBe('generic:binary.tar.gz');
        // No type row: the key begins with it.
        expect(lane.markers[0].details.map(d => d.label)).toEqual(['Created:', 'File:', 'Size:']);
        expect(lane.markers[0].details[2].value).toBe('2.0 kB');
    });

    it('pins a result on the step that was running when it was made', () => {
        // A result says which job made it, never which step — but both timestamps come off the same
        // worker, so the step covering the moment it was created is the step that created it.
        const job3 = job({
            ...runJob,
            steps_status: {
                checkout: <any>{ conclusion: 'Success', started: '2026-01-01T10:00:21Z', ended: '2026-01-01T10:00:30Z' },
                upload: <any>{ conclusion: 'Success', started: '2026-01-01T10:00:45Z', ended: '2026-01-01T10:00:58Z' }
            }
        });
        const built = buildRunTimeline(run, [job3], [result]);
        const lane = built.data.sections.flatMap(jobLanes)[0];

        // issued_at is 10:00:55, inside `upload`.
        expect(lane.lanes.map(l => l.label)).toEqual(['checkout', 'upload']);
        expect(lane.lanes[0].markers ?? []).toEqual([]);
        expect(lane.lanes[1].markers.length).toBe(1);
        expect(built.targets[lane.lanes[1].markers[0].id]).toEqual({ type: 'result', id: 'r1' });
    });


    it('pins a result on its job, and keeps a lane for one no step accounts for', () => {
        const built = buildRunTimeline(run, [runJob], [result]);
        const lane = built.data.sections.flatMap(jobLanes)[0];

        expect(lane.markers.length).toBe(1);
        expect(lane.markers[0].at).toBe(Date.parse(result.issued_at));
        expect(built.targets[lane.markers[0].id]).toEqual({ type: 'result', id: 'r1' });

        // The only step of this job ran 10:00:21 → 10:00:30 and the result was made at 10:00:55, so no
        // step accounts for it and it falls back to a lane of its own rather than being dropped.
        expect(lane.lanes.map(l => l.label)).toEqual(['checkout', 'binary.tar.gz']);
        expect(built.targets[lane.lanes[1].id]).toEqual({ type: 'result', id: 'r1' });
        expect(built.targets[lane.id]).toEqual({ type: 'job', id: 'j1' });
    });

    it('carries the dates the graph no longer shows, and nothing the view already says', () => {
        const retried = job({ ...runJob, retry: 2, worker_name: 'w-1', region: 'default' });
        const lane = buildRunTimeline(run, [retried], []).data.sections.flatMap(jobLanes)[0];

        // No status, no wall clock, no worker, no region: the first two are already in the colours and
        // the breakdown, and the last two are on the job panel.
        expect(lane.details.map(d => d.label)).toEqual(['Queued:', 'Scheduled:', 'Started:', 'Ended:', 'Retry:']);
        expect(lane.details.find(d => d.label === 'Retry:').value).toBe('2');
    });

    it('leaves the dates it was not given out of the detail', () => {
        const blocked = job({ id: 'j2', job_id: 'build', status: V2WorkflowRunJobStatus.Blocked, queued: '2026-01-01T10:00:01Z' });
        const lane = buildRunTimeline(run, [blocked], []).data.sections.flatMap(jobLanes)[0];

        expect(lane.details.map(d => d.label)).toEqual(['Queued:']);
    });

    it('starts the axis with the run on a first attempt', () => {
        expect(buildRunTimeline(run, [runJob], [], [], 1).data.start).toBe(Date.parse(run.started));
    });

    it('starts the axis with the jobs on a later attempt', () => {
        // A restart does not move the start of the run, so keeping it would open the timeline on a fold
        // spanning everything since the first attempt.
        expect(buildRunTimeline(run, [runJob], [], [], 2).data.start).toBeUndefined();
    });

    it('leaves out a job that never reached the queue', () => {
        const built = buildRunTimeline(run, [job({ id: 'j3', job_id: 'build', status: V2WorkflowRunJobStatus.Skipped })], []);

        // Nothing but the section of the run, which is there as soon as the run started.
        expect(built.data.sections.flatMap(jobLanes).length).toBe(0);
        expect(built.jobCount).toBe(0);
    });

    it('groups the jobs under their stage, in the order they were reached', () => {
        const first = job({ id: 'j1', job_id: 'build', queued: '2026-01-01T10:00:01Z', job: <any>{ stage: 'build' } });
        const second = job({ id: 'j2', job_id: 'deploy', queued: '2026-01-01T10:05:00Z', job: <any>{ stage: 'deploy' } });
        const built = buildRunTimeline(run, [second, first], []);

        // The run keeps a section of its own before them: it belongs to no stage, and putting its lane in
        // the first one said that it was part of it.
        expect(built.data.sections.map(s => s.id)[0]).toBe('run');
        expect(jobSections(built).map(s => s.label)).toEqual(['build', 'deploy']);
    });
});

describe('gates', () => {
    const run = <V2WorkflowRun>{
        started: '2026-01-01T10:00:00Z',
        workflow_data: {
            workflow: <any>{
                jobs: { deploy: { gate: 'manual' } },
                gates: { manual: { if: '${{ gate.manual }}', inputs: { version: { type: 'string' } } } }
            }
        }
    };

    it('marks the gate at the head of the job it held, with what was answered', () => {
        const triggered = job({
            id: 'j1', job_id: 'deploy', job: <any>{ gate: 'manual' },
            gate_inputs: { version: '1.4.2', force: true },
            queued: '2026-01-01T10:00:01Z', started: '2026-01-01T10:00:10Z', ended: '2026-01-01T10:00:40Z'
        });
        const lane = buildRunTimeline(run, [triggered], []).data.sections.flatMap(jobLanes)[0];

        expect(lane.markers.map(m => m.kind)).toEqual(['gate']);
        expect(lane.markers[0].at).toBe(Date.parse(triggered.queued));
        expect(lane.markers[0].label).toBe('Gate manual');
        // The values used are on the run job, the condition on the workflow.
        expect(lane.markers[0].details.map(d => `${d.label} ${d.value}`))
            .toEqual(['force: true', 'version: 1.4.2', 'Condition: ${{ gate.manual }}']);
    });

    it('says so when the gate has not been answered yet', () => {
        const waiting = job({
            id: 'j2', job_id: 'deploy', status: V2WorkflowRunJobStatus.Blocked,
            job: <any>{ gate: 'manual' }, queued: '2026-01-01T10:00:01Z'
        });
        const lane = buildRunTimeline(run, [waiting], []).data.sections.flatMap(jobLanes)[0];

        expect(lane.markers[0].details[0].label).toBe('Waiting to be triggered');
    });
});

describe('the chain of work', () => {
    // A skipped job is terminated and carries an end date like any other, but it did no work — so it has
    // no business being presented as having set the duration of anything.
    const chained = <V2WorkflowRun>{
        workflow_data: { workflow: <any>{ jobs: { a: {}, b: { needs: ['a'] }, c: { needs: ['b'] } } } }
    };

    it('steps over a skipped job to reach what did the work behind it', () => {
        const jobs = [
            job({ id: 'ja', job_id: 'a', ended: '2026-01-01T10:00:10Z' }),
            job({ id: 'jb', job_id: 'b', status: V2WorkflowRunJobStatus.Skipped, ended: '2026-01-01T10:00:11Z' }),
            job({ id: 'jc', job_id: 'c', ended: '2026-01-01T10:00:30Z' })
        ];

        expect(computeCriticalPath(chained, jobs)).toEqual(['jc', 'ja']);
    });

    it('never lets a skipped job head the chain, even when it ends last', () => {
        const jobs = [
            job({ id: 'ja', job_id: 'a', ended: '2026-01-01T10:00:10Z' }),
            job({ id: 'jb', job_id: 'b', status: V2WorkflowRunJobStatus.Skipped, ended: '2026-01-01T10:05:00Z' })
        ];

        expect(computeCriticalPath(chained, jobs)).toEqual(['ja']);
    });
});

describe('run errors on the timeline', () => {
    const run = <V2WorkflowRun>{
        started: '2026-01-01T10:00:00Z',
        workflow_name: 'build-and-deploy',
        workflow_data: { workflow: <any>{ jobs: { build: {} } } }
    };
    const runJob = job({
        id: 'j1', job_id: 'build', status: V2WorkflowRunJobStatus.Fail,
        queued: '2026-01-01T10:00:05Z', started: '2026-01-01T10:00:10Z', ended: '2026-01-01T10:00:40Z'
    });
    const infos = <Array<WorkflowRunInfo>>[
        { id: 'i1', issued_at: '2026-01-01T10:00:02Z', level: 'info', message: 'Workflow crafted' },
        { id: 'i2', issued_at: '2026-01-01T10:00:41Z', level: 'error', message: 'job build failed: exit status 1' },
        { id: 'i3', issued_at: '2026-01-01T10:00:42Z', level: 'warning', message: 'deprecated syntax' }
    ];

    it('marks what the run logged as an error, on a lane of the run itself', () => {
        const built = buildRunTimeline(run, [runJob], [], infos);

        // A run info belongs to the run and names no job, so it cannot go on a job's lane. The lane it
        // goes on is named for what it is rather than after the workflow, which read as a job name.
        expect(built.data.sections.map(s => s.lanes.map(l => l.label))).toEqual([['Run'], ['build']]);
        expect(runLane(built).sublabel).toBe('1 error');
        const error = runLane(built).markers.filter(m => m.kind === 'error');
        expect(error.map(m => m.label)).toEqual(['job build failed: exit status 1']);
        expect(error[0].at).toBe(Date.parse(infos[1].issued_at));
        // The Info tab holds the full text, so that is where activating it goes.
        expect(built.targets[error[0].id]).toEqual({ type: 'info', id: 'i2' });
        // The lane of the run is not one of its jobs.
        expect(built.jobCount).toBe(1);
    });

    it('leaves the ordinary chatter of a run out of it', () => {
        // Warnings and infos are the everyday noise; a mark for each would bury the one worth stopping at.
        const built = buildRunTimeline(run, [runJob], [], [infos[0], infos[2]]);

        // The lane of the run is still there, holding what set it off and nothing else.
        expect(runLane(built).markers.map(m => m.kind)).toEqual(['trigger']);
        expect(runLane(built).sublabel).toBeNull();
    });

    it('still shows the error of a run that queued no job at all', () => {
        // The run that failed while being crafted has nothing else to show, and is the one this matters for.
        const built = buildRunTimeline(run, [], [], [infos[1]]);

        expect(built.data.sections.length).toBe(1);
        expect(built.data.sections[0].id).toBe('run');
        // Two instants and not one bar, which is all such a run ever has: what started it and what
        // stopped it.
        expect(runLane(built).markers.map(m => m.kind)).toEqual(['trigger', 'error']);
        expect(built.jobCount).toBe(0);
    });

    it('gives errors their own word for when several are drawn as one', () => {
        const many = Array.from({ length: 3 }, (_, i) => <WorkflowRunInfo>{
            id: `e${i}`, issued_at: '2026-01-01T10:00:41Z', level: 'error', message: `failure ${i}`
        });
        const lane = runLane(buildRunTimeline(run, [runJob], [], many));

        // Without this a cluster of them would read "3 results", the word this timeline uses in general.
        expect([...new Set(lane.markers.filter(m => m.kind === 'error').map(m => m.plural))]).toEqual(['errors']);
        expect(lane.sublabel).toBe('3 errors');
    });
});

describe('what set the run off', () => {
    const pushed = <V2WorkflowRun>{
        started: '2026-01-01T10:00:00Z',
        workflow_name: 'build-and-deploy',
        username: 'someone',
        event: <any>{
            event_name: 'push', hook_type: 'RepositoryWebHook',
            ref: 'refs/heads/main', sha: '0123456789abcdef'
        },
        workflow_data: { workflow: <any>{ jobs: { build: {} } } }
    };

    it('marks the hook that fired, at the moment the run started', () => {
        const built = buildRunTimeline(pushed, [], []);
        const marker = runLane(built).markers[0];

        expect(marker.at).toBe(Date.parse(pushed.started));
        expect(marker.kind).toBe('trigger');
        expect(marker.label).toBe('Started by a push');
        // A branch reads as a branch, and eight characters of a sha are enough to recognise it.
        expect(marker.details.map(d => `${d.label} ${d.value}`.trim()).slice(0, 4))
            .toEqual(['Hook: Repository web hook', 'By: someone', 'Ref: main', 'Commit: 01234567']);
        // Activating it opens the hook panel, which is where the whole event is written out.
        expect(built.targets[marker.id]).toEqual({ type: 'hook', id: 'push' });
    });

    it('tells the kinds of trigger apart by their icon', () => {
        const iconOf = (event_name: string) => {
            const built = buildRunTimeline(<V2WorkflowRun>{ ...pushed, event: <any>{ event_name } }, [], []);
            return runLane(built).markers[0].icon;
        };

        expect(['manual', 'push', 'scheduler', 'webhook', 'workflow-run', 'model-update', ''].map(iconOf))
            .toEqual(['user', 'branches', 'clock-circle', 'global', 'appstore', 'sync', 'thunderbolt']);
    });

    it('has nothing to mark on a run with no start date', () => {
        const built = buildRunTimeline(<V2WorkflowRun>{ ...pushed, started: null }, [], []);

        expect(built.data.sections.length).toBe(0);
    });

    it('marks the restart instead on a later attempt, at the head of what it restarted', () => {
        // A restart two days after the run: the run was set off long before anything being shown, so
        // marking *it* there would drag the axis back to a moment that has nothing to do with this
        // attempt — the very thing starting the axis with the jobs of the attempt is there to avoid.
        const restarted = job({
            id: 'j1', job_id: 'build',
            queued: '2026-01-03T10:00:00Z', started: '2026-01-03T10:00:05Z', ended: '2026-01-03T10:01:00Z'
        });
        const again = <V2WorkflowRun>{
            ...pushed,
            run_attempt: 2,
            job_events: <any>[{ username: 'someone', job_id: 'build', run_attempt: 2, inputs: {} }]
        };

        const built = buildRunTimeline(again, [restarted], [], [], 2);
        const marker = runLane(built).markers[0];

        expect(marker.label).toBe('Restarted by someone');
        expect(marker.at).toBe(Date.parse(restarted.queued));
        expect(marker.details.map(d => d.label)).toEqual(['Job:', 'Attempt:', 'Started:']);
        // There is no panel for a restart, so like a gate it is a mark to hover and nothing more.
        expect(built.targets[marker.id]).toBeUndefined();

        // The same run read as its first attempt still marks the hook.
        expect(runLane(buildRunTimeline(again, [restarted], [], [], 1)).markers[0].label).toBe('Started by a push');
    });
});

describe('shouldFollow', () => {
    it('keeps up with a live run while nothing has been asked of the view', () => {
        expect(shouldFollow(true, false, false, 0)).toBeTrue();
        // Content shorter than the view leaves negative room, which is still the foot of it.
        expect(shouldFollow(true, false, false, -50)).toBeTrue();
    });

    it('leaves the view alone the moment anything is asked of it', () => {
        // Scrolled up to read something further back.
        expect(shouldFollow(true, false, false, 200)).toBeFalse();
        // Zoomed in on something.
        expect(shouldFollow(true, true, false, 0)).toBeFalse();
        // Pointing at something.
        expect(shouldFollow(true, false, true, 0)).toBeFalse();
    });

    it('has nothing to keep up with once the run has finished', () => {
        expect(shouldFollow(false, false, false, 0)).toBeFalse();
    });

    it('picks it up again on the way back down, having remembered nothing', () => {
        expect(shouldFollow(true, false, false, 300)).toBeFalse();
        expect(shouldFollow(true, false, false, 4)).toBeTrue();
    });
});

describe('revealBy', () => {
    const view = { top: 100, bottom: 400 };

    it('leaves a lane already in view where it is', () => {
        expect(revealBy(view, { top: 200, bottom: 220 }, 10)).toBe(0);
        // Right up against either edge, margin included.
        expect(revealBy(view, { top: 110, bottom: 390 }, 10)).toBe(0);
    });

    it('scrolls up by the least that shows a lane above the view', () => {
        expect(revealBy(view, { top: 60, bottom: 80 }, 10)).toBe(-50);
        // In view but for its top edge: only the missing part, and the margin, is scrolled.
        expect(revealBy(view, { top: 95, bottom: 200 }, 10)).toBe(-15);
    });

    it('scrolls down by the least that shows a lane below the view', () => {
        expect(revealBy(view, { top: 500, bottom: 520 }, 10)).toBe(130);
        expect(revealBy(view, { top: 300, bottom: 405 }, 10)).toBe(15);
    });

    it('shows a lane taller than the view from its top rather than its foot', () => {
        // Scrolling to its foot would have carried its name off the top of the view.
        expect(revealBy(view, { top: 500, bottom: 900 }, 10)).toBe(390);
    });
});
