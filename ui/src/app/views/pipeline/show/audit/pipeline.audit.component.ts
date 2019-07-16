import { Component, Input, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Column, ColumnType } from 'app/shared/table/data-table.component';
import { FetchPipelineAudits, RollbackPipeline } from 'app/store/pipelines.action';
import { compare } from 'fast-json-patch';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize, first } from 'rxjs/operators';
import { Action } from '../../../../model/action.model';
import { Job } from '../../../../model/job.model';
import { Pipeline, PipelineAudit, PipelineAuditDiff } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { Stage } from '../../../../model/stage.model';
import { Item } from '../../../../shared/diff/list/diff.list.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-pipeline-audit',
    templateUrl: './pipeline.audit.html',
    styleUrls: ['./pipeline.audit.scss']
})
export class PipelineAuditComponent implements OnInit {
    @Input() project: Project;
    @Input() pipeline: Pipeline;

    currentCompare: Array<PipelineAuditDiff>;
    items: Array<Item>;

    indexSelected: number;
    codeMirrorConfig: any;
    loading = false;
    columns: Column<PipelineAudit>[];

    constructor(
        private store: Store,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: 'nocursor'
        };
    }

    ngOnInit(): void {
        this.loading = true;
        this.store.dispatch(new FetchPipelineAudits({
            projectKey: this.project.key,
            pipelineName: this.pipeline.name
        })).pipe(finalize(() => this.loading = false))
            .subscribe();

        this.columns = [
            <Column<PipelineAudit>>{
                type: ColumnType.TEXT,
                name: 'audit_action',
                selector: (audit: PipelineAudit) => audit.action,
            },
            <Column<PipelineAudit>>{
                type: ColumnType.TEXT,
                name: 'audit_username',
                selector: (audit: PipelineAudit) => audit.username,
            },
            <Column<PipelineAudit>>{
                type: ColumnType.DATE,
                name: 'audit_time_author',
                selector: (audit: PipelineAudit) => audit.versionned,
            },
            <Column<PipelineAudit>>{
                type: ColumnType.CONFIRM_BUTTON,
                name: '',
                selector: (audit: PipelineAudit) => {
                    return {
                        title: 'common_rollback',
                        click: () => this.rollback(audit.id)
                    };
                },
            },
        ];
    }

    compareIndex(audit: PipelineAudit): void {
        let pipFrom = cloneDeep(audit.pipeline);

        if (!this.pipeline.audits) {
            return;
        }

        let pipFromIdx = this.pipeline.audits.findIndex((aud) => aud.id === audit.id);
        if (pipFromIdx < 0) {
            return;
        }
        let pipTo: Pipeline;
        if (pipFromIdx === 0) {
            pipTo = cloneDeep(this.pipeline);
        } else {
            pipTo = cloneDeep(this.pipeline.audits[pipFromIdx - 1].pipeline);
        }

        pipFrom = this.cleanPipeline(pipFrom);
        pipTo = this.cleanPipeline(pipTo);

        this.currentCompare = new Array<PipelineAuditDiff>();
        compare(pipFrom, pipTo).forEach(c => {
            let diff: PipelineAuditDiff = null;
            let path = c.path;
            let pathSplitted = path.split('/').filter(p => p !== '');

            switch (audit.action) {
                case 'addStage':
                    diff = this.getAddStageDiff(pathSplitted, pipTo);
                    break;
                case 'updateStage':
                    diff = this.getUpdateStageDiff(pathSplitted, pipTo, pipFrom);
                    break;
                case 'deleteStage':
                    diff = this.getDeleteStageDiff(pathSplitted, pipFrom);
                    break;
                case 'addJob':
                    diff = this.getAddJobDiff(pathSplitted, pipTo);
                    break;
                case 'updateJob':
                    diff = this.getUpdateJobDiff(path, pathSplitted, pipTo, pipFrom);
                    break;
                case 'deleteJob':
                    diff = this.getDeleteJobDiff(pathSplitted, pipFrom);
                    break;
            }

            if (diff) {
                this.currentCompare.push(diff);
            }
        });

        this.items = this.currentCompare.map(c => {
            return <Item>{
                name: c.title,
                before: c.before,
                after: c.after,
                type: c.type === 'json' ? 'application/json' : 'text/plain'
            }
        });
    }

    getAddJobDiff(path: Array<string>, pipTo: Pipeline): PipelineAuditDiff {
        let diff = new PipelineAuditDiff();
        let jobIndex = 0;
        if (path.length > 3) {
            jobIndex = Number(path[3]);

        }
        diff.title = 'Add ' + pipTo[path[0]][path[1]].name + ' > ' + pipTo[path[0]][path[1]][path[2]][jobIndex].action.name;
        diff.after = JSON.stringify(this.cleanJob(pipTo[path[0]][path[1]][path[2]][jobIndex]), undefined, 4);
        diff.type = 'json';
        diff.before = null;
        return diff;
    }

    getAddStageDiff(path: Array<string>, pipTo: Pipeline): PipelineAuditDiff {
        let diff = new PipelineAuditDiff();
        diff.title = 'Add ' + pipTo[path[0]][path[1]].name;
        diff.type = 'json';

        diff.after = JSON.stringify(this.cleanStage(pipTo[path[0]][path[1]]), undefined, 4);
        diff.before = null;
        return diff;
    }

    cleanPipeline(p: Pipeline): Pipeline {
        delete p.last_modified;
        if (p.usage) {
            delete p.usage.applications;
        }
        return p;
    }

    cleanStage(s: Stage): Stage {
        delete s.id;
        delete s.build_order;
        delete s.run_jobs;
        delete s.last_modified;
        delete s.status;
        delete s.warnings;

        if (s.jobs) {
            for (let i = 0; i < s.jobs.length; i++) {
                s.jobs[i] = this.cleanJob(s.jobs[i]);
            }
        }
        return s;
    }

    cleanJob(j: Job): Job {
        delete j.warnings;
        delete j.last_modified;
        delete j.pipeline_action_id;
        delete j.hasChanged;
        delete j.step_status;
        delete j.pipeline_stage_id;

        delete j.action.hasChanged;
        delete j.action.type;
        delete j.action.id;
        delete j.action.loading;

        j.action.actions = this.cleanSteps(j.action.actions);
        return j;
    }

    cleanSteps(steps: Array<Action>) {
        if (steps) {
            for (let i = 0; i < steps.length; i++) {
                delete steps[i].id;
                delete steps[i].requirements;
                delete steps[i].description;
                delete steps[i].type;
                delete steps[i].actions;
                if (steps[i].parameters) {
                    for (let k = 0; k < steps[i].parameters.length; k++) {
                        delete steps[i].parameters[k].id;
                        delete steps[i].parameters[k].type;
                        delete steps[i].parameters[k].description;
                    }
                }
            }
        }
        return steps;
    }

    getDeleteStageDiff(path: Array<string>, pipFrom: Pipeline): PipelineAuditDiff {
        let diff = new PipelineAuditDiff();
        diff.title = 'Delete ' + pipFrom[path[0]][path[1]].name;
        diff.type = 'json';
        diff.before = JSON.stringify(this.cleanStage(pipFrom[path[0]][path[1]]), undefined, 4);
        diff.after = null;
        return diff;
    }

    getDeleteJobDiff(path: Array<string>, pipFrom: Pipeline): PipelineAuditDiff {
        let diff = new PipelineAuditDiff();
        let jobIndex = 0;
        if (path.length > 3) {
            jobIndex = Number(path[3]);
        }
        diff.title = 'Remove ' + pipFrom[path[0]][path[1]].name + ' > ' + pipFrom[path[0]][path[1]][path[2]][jobIndex].action.name;
        diff.before = JSON.stringify(this.cleanJob(pipFrom[path[0]][path[1]][path[2]][jobIndex]), undefined, 4);
        diff.type = 'json';
        diff.after = null;
        return diff;
    }

    getUpdateJobDiff(path: string, pathSplitted: Array<string>, pipTo: Pipeline, pipFrom: Pipeline): PipelineAuditDiff {
        let diff = new PipelineAuditDiff();
        if (!pathSplitted.length || pathSplitted.length < 2) {
            return;
        }

        let stage: Stage = pipTo[pathSplitted[0]][pathSplitted[1]];
        let job: Job = new Job();

        if (pathSplitted.length > 3) {
            job = stage.jobs[pathSplitted[3]];
        }

        if (path.indexOf('requirements') !== -1) {
            diff.title = 'Update ' + stage.name + ' > ' + job.action.name + ' > requirements';
            if (!pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]]) {
                return null;
            }
            diff.before = JSON.stringify(pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]].action.requirements, undefined, 4);
            diff.after = JSON.stringify(job.action.requirements, undefined, 4);
            diff.type = 'json';
        } else if (path.indexOf('actions') !== -1) {
            if (path.indexOf('always_executed') !== -1 || path.indexOf('optional') !== -1 || path.indexOf('enabled') !== -1) {
                diff.title = 'Update ' + stage.name + ' > ' + job.action.name + ' > steps > '
                    + job.action.actions[pathSplitted[6]].name + ' > ' + pathSplitted[7];
                if (!pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]]) {
                    return null;
                }
                diff.before = pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]].action.actions[pathSplitted[6]][pathSplitted[7]];
                diff.after = job.action.actions[pathSplitted[6]][pathSplitted[7]];
                diff.type = 'string';
            } else {
                diff.title = 'Update ' + stage.name + ' > ' + job.action.name + ' > steps';
                if (!pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]]) {
                    return null;
                }
                diff.before = JSON.stringify(
                    this.cleanSteps(pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]].action.actions), undefined, 4);
                diff.after = JSON.stringify(this.cleanSteps(job.action.actions), undefined, 4);
                diff.type = 'json';
            }
        } else if (pathSplitted.length === 5 && pathSplitted[4] === 'enabled') {
            return null;
        } else {
            // change enabled/description/name
            diff.title = 'Update ' + stage.name + ' > ' + job.action.name + ' > ' + pathSplitted[5];
            diff.type = 'string';
            if (!pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]]) {
                return null;
            }
            diff.before = pipFrom.stages[pathSplitted[1]].jobs[pathSplitted[3]].action[pathSplitted[5]];
            diff.after = job.action[pathSplitted[5]];
        }
        return diff;
    }

    getUpdateStageDiff(path: Array<string>, pipTo: Pipeline, pipFrom: Pipeline): PipelineAuditDiff {
        let diff = new PipelineAuditDiff();

        if (path.length === 3 && (path[2] === 'enabled' || path[2] === 'name')) {
            diff.type = 'string';
            diff.after = pipTo[path[0]][path[1]][path[2]];
            diff.before = pipFrom[path[0]][path[1]][path[2]];
            diff.title = 'Update ' + pipTo[path[0]][path[1]].name + ' > ' + path[2];
        } else if (path.length === 3 && path[2] === 'prerequisites') {
            // add first prerequisite or delete last prerequisite
            if (!pipTo[path[0]][path[1]][path[2]] || pipTo[path[0]][path[1]][path[2]].length === 0) {
                diff.title = 'Remove ' + pipTo[path[0]][path[1]].name + ' > prerequisite';
                diff.before = JSON.stringify(pipFrom[path[0]][path[1]][path[2]], undefined, 4);
                diff.after = null;
                diff.type = 'json';
            } else {
                diff.title = 'Add ' + pipTo[path[0]][path[1]].name + ' > prerequisite';
                diff.after = JSON.stringify(pipTo[path[0]][path[1]][path[2]], undefined, 4);
                diff.before = null;
                diff.type = 'json';
            }
        } else if (path.length > 3 && path[2] === 'prerequisites') {
            diff.title = 'Update ' + pipTo[path[0]][path[1]].name + ' > prerequisite';
            diff.before = JSON.stringify(pipFrom[path[0]][path[1]][path[2]], undefined, 4);
            diff.after = JSON.stringify(pipTo[path[0]][path[1]][path[2]], undefined, 4);
            diff.type = 'json';
        } else {
            return null;
        }

        return diff;
    }

    rollback(auditId: number): void {
        this.loading = true;
        this.store.dispatch(new RollbackPipeline({
            projectKey: this.project.key,
            pipelineName: this.pipeline.name,
            auditId
        })).pipe(
            first(),
            finalize(() => this.loading = false)
        ).subscribe(() => this._toast.success('', this._translate.instant('pipeline_updated')));
    }
}
