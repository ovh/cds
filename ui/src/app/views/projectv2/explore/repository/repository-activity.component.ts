import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { lastValueFrom, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { HookEventWorkflowStatus, RepositoryHookEvent, RepositoryHookWorkflow } from 'app/model/project.model';
import { RepositoryAnalysis } from 'app/model/analysis.model';
import { ProjectService } from 'app/service/project/project.service';
import { apiErrorMessage, RepositoryContextService } from './repository-context.service';
import { cleanErrorMessage, eventAuthor, HookEventVerdict, hookEventVerdict, StepStatus, VerdictLevel } from './hook-event-verdict';
import { AnalysisSummaryPart, analysisSummary } from './analysis-outcome';

/** One event of the repository, with what the page says about it. */
interface ActivityRow {
    event: RepositoryHookEvent;
    verdict: HookEventVerdict;
    created: string;
    /** The analysis this project ran for the event, when it did. */
    analysisId: string;
}

const PROBLEM_LEVELS: Array<VerdictLevel> = ['error', 'warning'];

/**
 * The events a repository received, one verdict per line, and step by step when a line is opened.
 * The analysis of an opened event is read on demand, once.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository-activity',
    templateUrl: './repository-activity.html',
    styleUrls: ['./repository-activity.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryActivityComponent implements OnInit, OnDestroy {
    ctx = inject(RepositoryContextService);
    /** All the events, most recent first; null until read. */
    rows: Array<ActivityRow> = null;
    filtered: Array<ActivityRow> = [];
    problemsOnly: boolean = false;
    eventFilter: string = null;
    eventNames: Array<string> = [];
    expanded = new Set<string>();
    analyses = new Map<string, RepositoryAnalysis | 'loading' | 'error'>();
    loading: boolean = false;
    distant: boolean = false;
    readonly scopes = [{ label: 'All', value: 'all' }, { label: 'Problems only', value: 'problems' }];

    eventsSub: Subscription;
    repositorySub: Subscription;

    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.repositorySub = this.ctx.repository$.subscribe(repository => {
            this.distant = !!repository?.distant;
        });
        this.eventsSub = this.ctx.events$.subscribe(events => {
            this.buildRows(events);
            this._cd.markForCheck();
        });
    }

    private buildRows(events: Array<RepositoryHookEvent>): void {
        if (events === null) {
            this.rows = null;
            this.filtered = [];
            return;
        }
        const key = this.ctx.project.key;
        this.rows = events
            .map(event => ({
                event,
                verdict: hookEventVerdict(event, key, this.distant),
                created: new Date(event.created / 1000000).toJSON(),
                analysisId: (event.analyses ?? []).find(a => a.project_key === key)?.analyze_id ?? null
            }))
            .sort((a, b) => b.event.created - a.event.created);
        this.eventNames = [...new Set(this.rows.map(r => String(r.event.event_name)))].sort();
        this.applyFilters();
    }

    applyFilters(): void {
        this.filtered = (this.rows ?? [])
            .filter(r => !this.problemsOnly || PROBLEM_LEVELS.indexOf(r.verdict.level) !== -1)
            .filter(r => !this.eventFilter || String(r.event.event_name) === this.eventFilter);
    }

    changeScope(scope: string): void {
        this.problemsOnly = scope === 'problems';
        this.applyFilters();
        this._cd.markForCheck();
    }

    changeEventFilter(name: string): void {
        this.eventFilter = name || null;
        this.applyFilters();
        this._cd.markForCheck();
    }

    async refresh() {
        this.loading = true;
        this._cd.markForCheck();
        await this.ctx.reloadEvents();
        this.loading = false;
        this._cd.markForCheck();
    }

    toggle(row: ActivityRow): void {
        if (this.expanded.has(row.event.uuid)) {
            this.expanded.delete(row.event.uuid);
        } else {
            this.expanded.add(row.event.uuid);
            this.loadAnalysis(row);
        }
        this._cd.markForCheck();
    }

    isExpanded(row: ActivityRow): boolean {
        return this.expanded.has(row.event.uuid);
    }

    /** The analysis of an opened event is read once, whatever the number of times it is opened. */
    private async loadAnalysis(row: ActivityRow) {
        if (!row.analysisId || this.analyses.has(row.analysisId)) {
            return;
        }
        this.analyses.set(row.analysisId, 'loading');
        try {
            const analysis = await lastValueFrom(this._projectService.getAnalysis(this.ctx.project.key, row.event.vcs_server_name, row.event.repository_name, row.analysisId));
            this.analyses.set(row.analysisId, analysis);
        } catch (e) {
            this.analyses.set(row.analysisId, 'error');
        }
        this._cd.markForCheck();
    }

    analysisOf(row: ActivityRow): RepositoryAnalysis {
        const analysis = row.analysisId ? this.analyses.get(row.analysisId) : null;
        return analysis && analysis !== 'loading' && analysis !== 'error' ? analysis : null;
    }

    analysisLoading(row: ActivityRow): boolean {
        return !!row.analysisId && this.analyses.get(row.analysisId) === 'loading';
    }

    analysisSummary(analysis: RepositoryAnalysis): Array<AnalysisSummaryPart> {
        return analysisSummary(analysis);
    }

    /** The analyses tab of the repository, where an analysis is read in full. */
    get analysesLink(): Array<string> {
        return ['/project', this.ctx.project.key, 'explore', 'vcs', this.ctx.vcsName, 'repository', this.ctx.repoName, 'analyses'];
    }

    /** The run a workflow started, when it lives in this project. */
    runLink(w: RepositoryHookWorkflow): Array<string> {
        return w.run_id && w.project_key === this.ctx.project.key ? ['/project', w.project_key, 'run', w.run_id] : null;
    }

    /** Why a workflow shows no run, in a few words; nothing for a triggered one. */
    workflowNote(w: RepositoryHookWorkflow): string {
        switch (w.status) {
            case HookEventWorkflowStatus.Scheduled: return 'scheduled';
            case HookEventWorkflowStatus.Error: return w.error ? `did not start: ${cleanErrorMessage(w.error)}` : 'did not start';
            case HookEventWorkflowStatus.Skipped: return w.error ? `skipped: ${w.error}` : 'skipped';
            default: return null;
        }
    }

    levelIcon(level: VerdictLevel): string {
        switch (level) {
            case 'success': return 'check-circle';
            case 'error': return 'close-circle';
            case 'warning': return 'exclamation-circle';
            case 'processing': return 'sync';
            case 'filtered': return 'stop';
            default: return 'minus-circle';
        }
    }

    /** The icon of a step, in the same palette as the row: green done, red failed, orange partly, blue running, grey otherwise. */
    stepIcon(status: StepStatus): string {
        switch (status) {
            case 'finish': return 'check-circle';
            case 'error': return 'close-circle';
            case 'warning': return 'exclamation-circle';
            case 'process': return 'sync';
            case 'pending': return 'clock-circle';
            case 'skipped': return 'stop';
            default: return 'minus-circle';
        }
    }

    /**
     * What Ant Design knows of a step. Never `error`: it would draw its own cross next to ours; the
     * red of a failed step comes from a class of ours instead.
     */
    stepAntStatus(status: StepStatus): string {
        switch (status) {
            case 'finish': case 'warning': case 'error': return 'finish';
            case 'process': return 'process';
            default: return 'wait';
        }
    }

    tagColor(level: VerdictLevel): string {
        switch (level) {
            case 'success': return 'success';
            case 'error': return 'error';
            case 'warning': return 'warning';
            case 'processing': return 'processing';
            default: return 'default';
        }
    }

    author(row: ActivityRow): string {
        return eventAuthor(row.event, this.ctx.project.key);
    }

    errorMessage(e: any): string {
        return apiErrorMessage(e);
    }
}
