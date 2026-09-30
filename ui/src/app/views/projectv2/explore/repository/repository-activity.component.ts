import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { lastValueFrom, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { HookEventWorkflowStatus, RepositoryHookEvent, RepositoryHookWorkflow } from 'app/model/project.model';
import { RepositoryAnalysis } from 'app/model/analysis.model';
import { OperationStatus } from 'app/model/workflow-template.model';
import { ProjectService } from 'app/service/project/project.service';
import { apiErrorMessage, RepositoryContextService } from './repository-context.service';
import { cleanErrorMessage, eventAuthor, HookEventVerdict, hookEventVerdict } from './hook-event-verdict';
import { AnalysisSummaryPart, analysisSummary } from './analysis-outcome';
import { Tone, toneColor, toneIcon } from './palette';

/** One event of the repository, with what the page says about it. */
interface ActivityRow {
    event: RepositoryHookEvent;
    verdict: HookEventVerdict;
    created: string;
    /** The analysis this project ran for the event, when it did. */
    analysisId: string;
}

const PROBLEM_LEVELS: Array<Tone> = ['error', 'warning'];

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
    readonly icon = toneIcon;
    readonly color = toneColor;

    eventsSub: Subscription;
    repositorySub: Subscription;
    querySub: Subscription;

    /** The event the url asks to open, until it is shown. */
    private _wanted: string = null;
    private _route = inject(ActivatedRoute);
    private _router = inject(Router);

    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.repositorySub = this.ctx.repository$.subscribe(repository => {
            this.distant = !!repository?.distant;
        });
        this.eventsSub = this.ctx.events$.subscribe(events => {
            this.buildRows(events);
            this.revealWanted();
            this._cd.markForCheck();
        });
        this.querySub = this._route.queryParamMap.subscribe(params => {
            this._wanted = params.get('event');
            this.revealWanted();
        });
    }

    /** Opens the event the url names once it is listed, then drops the parameter so that it does not follow to the other tabs. */
    private revealWanted(): void {
        if (!this._wanted || !this.rows?.some(r => r.event.uuid === this._wanted)) {
            return;
        }
        const uuid = this._wanted;
        this._wanted = null;
        this.expanded.add(uuid);
        this._cd.markForCheck();
        setTimeout(() => document.getElementById(`event-${uuid}`)?.scrollIntoView({ block: 'center' }));
        this._router.navigate([], { relativeTo: this._route, queryParams: { event: null }, queryParamsHandling: 'merge', replaceUrl: true });
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

    changeProblemsOnly(problemsOnly: boolean): void {
        this.problemsOnly = problemsOnly;
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
        return this.ctx.repositoryLink('analyses');
    }

    /** The run a workflow started, when it lives in this project. */
    runLink(w: RepositoryHookWorkflow): Array<string> {
        return w.run_id && w.project_key === this.ctx.project.key ? ['/project', w.project_key, 'run', w.run_id] : null;
    }

    /**
     * Why a workflow shows no run, in a few words; nothing for a triggered one. The red says a failed
     * one did not start. One still scheduled once the event is settled never will: its git information
     * could not be read, or the event failed before reaching it.
     */
    workflowNote(row: ActivityRow, w: RepositoryHookWorkflow): string {
        switch (w.status) {
            case HookEventWorkflowStatus.Scheduled:
                if (row.verdict.level === 'processing') {
                    return 'scheduled';
                }
                return w.operation_status === OperationStatus.ERROR
                    ? `git information unavailable${w.operation_error ? ': ' + cleanErrorMessage(w.operation_error) : ''}`
                    : 'not started';
            case HookEventWorkflowStatus.Error: return cleanErrorMessage(w.error) || 'did not start';
            case HookEventWorkflowStatus.Skipped: return w.error ? `skipped: ${w.error}` : 'skipped';
            default: return null;
        }
    }

    /**
     * What Ant Design knows of a step. Never `error`: it would draw its own cross next to ours; the
     * red of a failed step comes from a class of ours instead.
     */
    stepAntStatus(status: Tone): string {
        switch (status) {
            case 'success': case 'warning': case 'error': return 'finish';
            case 'processing': return 'process';
            default: return 'wait';
        }
    }

    author(row: ActivityRow): string {
        return eventAuthor(row.event, this.ctx.project.key);
    }

    errorMessage(e: any): string {
        return apiErrorMessage(e);
    }
}
