import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { lastValueFrom, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { RepositoryHookEvent, RepositoryHookWorkflow } from 'app/model/project.model';
import { DataEntity, RepositoryAnalysis } from 'app/model/analysis.model';
import { EntityTypeUtil } from 'app/model/entity.model';
import { ProjectService } from 'app/service/project/project.service';
import { apiErrorMessage, RepositoryContextService } from './repository-context.service';
import { eventAuthor, HookEventVerdict, hookEventVerdict, VerdictLevel } from './hook-event-verdict';
import { entityOfAnalysisFile } from './entities';

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
    /** Decided from the first events: problems first when there are some. */
    problemsOnly: boolean = null;
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
        if (this.problemsOnly === null) {
            this.problemsOnly = this.rows.some(r => PROBLEM_LEVELS.indexOf(r.verdict.level) !== -1);
        }
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

    /** The files an analysis did not register: the ones worth a look. */
    rejectedFiles(analysis: RepositoryAnalysis): Array<DataEntity> {
        return (analysis.data?.entities ?? []).filter(e => e.status !== 'Success');
    }

    updatedCount(analysis: RepositoryAnalysis): number {
        return (analysis.data?.entities ?? []).filter(e => e.status === 'Success').length;
    }

    /** Where a file of the analysis is read in this page, on the ref of the event. */
    fileLink(file: DataEntity): Array<string> {
        const entity = entityOfAnalysisFile(file.path, file.file_name);
        if (!entity) {
            return null;
        }
        return ['/project', this.ctx.project.key, 'explore', 'vcs', this.ctx.vcsName, 'repository', this.ctx.repoName, EntityTypeUtil.toURLParam(entity.type), entity.name];
    }

    triggeredWorkflows(row: ActivityRow): Array<RepositoryHookWorkflow> {
        return (row.event.workflows ?? []).filter(w => w.project_key === this.ctx.project.key && w.run_id);
    }

    levelIcon(level: VerdictLevel): string {
        switch (level) {
            case 'success': return 'check-circle';
            case 'error': return 'close-circle';
            case 'warning': return 'exclamation-circle';
            case 'processing': return 'sync';
            default: return 'minus-circle';
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
