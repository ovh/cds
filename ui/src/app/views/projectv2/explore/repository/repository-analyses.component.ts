import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DataEntity, RepositoryAnalysis } from 'app/model/analysis.model';
import { EntityTypeUtil } from 'app/model/entity.model';
import { RepositoryContextService, shortRef } from './repository-context.service';
import { entityOfAnalysisFile, plural } from './entities';
import { analysisFileLabel, analysisOutcome } from './analysis-outcome';
import { Tone, toneColor, toneIcon } from './palette';

/** One analysis of the repository, with what the page says about it. */
interface AnalysisRow {
    analysis: RepositoryAnalysis;
    /** Every file the analysis read, registered or not, in the order it reports them. */
    files: Array<DataEntity>;
    registered: number;
    rejected: number;
    requestedBy: string;
    /** Asked for by hand rather than started by a repository event. */
    manual: boolean;
    outcome: Tone;
    /** The outcome in a few words, and what explains it when something went wrong. */
    label: string;
    detail: string;
}

/**
 * The analyses of the repository: each read of a ref's `.cds/` folder, what it registered and what
 * it rejected. This is where a definition that does not show up in the entity tabs gets explained.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository-analyses',
    templateUrl: './repository-analyses.html',
    styleUrls: ['./repository-analyses.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryAnalysesComponent implements OnInit, OnDestroy {
    ctx = inject(RepositoryContextService);
    /** All the analyses, most recent first; null until read. */
    rows: Array<AnalysisRow> = null;
    filtered: Array<AnalysisRow> = [];
    refs: Array<string> = [];
    refFilter: string = null;
    statusFilter: string = null;
    expanded = new Set<string>();
    loading: boolean = false;
    readonly icon = toneIcon;
    readonly color = toneColor;

    analysesSub: Subscription;
    querySub: Subscription;

    /** The analysis the url asks to open, until it is shown. */
    private _wanted: string = null;
    private _cd = inject(ChangeDetectorRef);
    private _route = inject(ActivatedRoute);
    private _router = inject(Router);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.analysesSub = this.ctx.analyses$.subscribe(analyses => {
            this.buildRows(analyses);
            this.revealWanted();
            this._cd.markForCheck();
        });
        this.querySub = this._route.queryParamMap.subscribe(params => {
            this._wanted = params.get('analysis');
            this.revealWanted();
        });
    }

    /** Opens the analysis the url names once it is listed, then drops the parameter so that it does not follow to the other tabs. */
    private revealWanted(): void {
        if (!this._wanted || !this.rows?.some(r => r.analysis.id === this._wanted)) {
            return;
        }
        const id = this._wanted;
        this._wanted = null;
        this.expanded.add(id);
        this._cd.markForCheck();
        setTimeout(() => document.getElementById(`analysis-${id}`)?.scrollIntoView({ block: 'center' }));
        this._router.navigate([], { relativeTo: this._route, queryParams: { analysis: null }, queryParamsHandling: 'merge', replaceUrl: true });
    }

    private buildRows(analyses: Array<RepositoryAnalysis>): void {
        if (analyses === null) {
            this.rows = null;
            this.filtered = [];
            return;
        }
        this.rows = analyses
            .map(analysis => {
                const files = analysis.data?.entities ?? [];
                const registered = files.filter(e => e.status === 'Success').length;
                const rejected = files.length - registered;
                const outcome = analysisOutcome(analysis.status, analysis.data?.error);
                return {
                    analysis,
                    files,
                    registered,
                    rejected,
                    requestedBy: this.requestedBy(analysis),
                    manual: !analysis.data?.hook_event_uuid,
                    outcome,
                    label: this.label(analysis, outcome, registered, rejected),
                    detail: outcome === 'success' ? null : analysis.data?.error || null
                };
            })
            .sort((a, b) => Date.parse(b.analysis.created) - Date.parse(a.analysis.created));
        this.refs = [...new Set(this.rows.map(r => r.analysis.ref))].sort();
        this.applyFilters();
    }

    private label(analysis: RepositoryAnalysis, outcome: Tone, registered: number, rejected: number): string {
        switch (outcome) {
            case 'success': return `${registered} ${plural(registered, 'file')} processed`;
            case 'warning': return `${rejected} ${plural(rejected, 'file')} skipped`;
            case 'error': return 'Analysis failed';
            case 'processing': return 'In progress';
            default: return analysis.status === 'Skipped' ? 'Nothing to register' : analysis.status;
        }
    }

    /** Who asked: the CDS user of the initiator, else its vcs user, else what older analyses carry. */
    private requestedBy(analysis: RepositoryAnalysis): string {
        const initiator = analysis.data?.initiator;
        return initiator?.user?.username || initiator?.vcs_username || analysis.data?.cds_username || null;
    }

    applyFilters(): void {
        this.filtered = (this.rows ?? [])
            .filter(r => !this.refFilter || r.analysis.ref === this.refFilter)
            .filter(r => !this.statusFilter || r.analysis.status === this.statusFilter);
    }

    changeRefFilter(ref: string): void {
        this.refFilter = ref || null;
        this.applyFilters();
        this._cd.markForCheck();
    }

    changeStatusFilter(status: string): void {
        this.statusFilter = status || null;
        this.applyFilters();
        this._cd.markForCheck();
    }

    async refresh() {
        this.loading = true;
        this._cd.markForCheck();
        await this.ctx.reloadAnalyses();
        this.loading = false;
        this._cd.markForCheck();
    }

    toggle(row: AnalysisRow): void {
        if (this.expanded.has(row.analysis.id)) {
            this.expanded.delete(row.analysis.id);
        } else {
            this.expanded.add(row.analysis.id);
        }
        this._cd.markForCheck();
    }

    isExpanded(row: AnalysisRow): boolean {
        return this.expanded.has(row.analysis.id);
    }

    /** The activity tab of the repository, where the event that started an analysis is read in full. */
    get activityLink(): Array<string> {
        return this.ctx.repositoryLink('activity');
    }

    /** Where a file of the analysis is read in this page, on the ref of the analysis. */
    fileLink(file: DataEntity): Array<string> {
        // Only a registered file has a page: a rejected one never made it to CDS
        const entity = file.status === 'Success' ? entityOfAnalysisFile(file.path, file.file_name) : null;
        if (!entity) {
            return null;
        }
        return this.ctx.repositoryLink(EntityTypeUtil.toURLParam(entity.type), entity.name);
    }

    /**
     * The files worth listing: all of them for an analysis that went through, only the ones left
     * unregistered for one that did not.
     */
    filesToShow(row: AnalysisRow): Array<DataEntity> {
        return row.analysis.status === 'Error' ? row.files.filter(f => f.status !== 'Success') : row.files;
    }

    /** A file's fate, as a badge: registered, refused for lack of permission, or never reached. */
    fileBadge(file: DataEntity): string {
        switch (file.status) {
            case 'Success': return 'success';
            case 'Skipped': return 'warning';
            default: return 'default';
        }
    }

    /** Only a file that was not taken needs a word; the green dot says enough for the others. */
    fileLabel(file: DataEntity): string {
        return analysisFileLabel(file);
    }

    shortRef(ref: string): string {
        return shortRef(ref);
    }
}
