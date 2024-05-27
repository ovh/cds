import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, ViewChild } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Project, ProjectRepository } from "app/model/project.model";
import { Store } from "@ngxs/store";
import { ActivatedRoute } from "@angular/router";
import { ProjectService } from "app/service/project/project.service";
import { ProjectState } from "app/store/project.state";
import { forkJoin } from "rxjs";
import { finalize } from "rxjs/operators";
import { Entity, EntityType } from "app/model/entity.model";
import { NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { Schema } from "app/model/json-schema.model";
import { VCSProject } from "app/model/vcs.model";

@Component({
    selector: 'app-projectv2-workflow-show',
    templateUrl: './project.workflow.show.html',
    styleUrls: ['./project.workflow.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowShowComponent implements OnDestroy {
    @ViewChild('editor') editor: NzCodeEditorComponent;

    loading: boolean;
    project: Project;
    vcsProject: VCSProject;
    repository: ProjectRepository;
    workflow: Entity;
    workflowJsonSchema: Schema;
    jobSchema: Schema;
    currentWorkflowName: string;
    currentBranch: string;
    errorNotFound: boolean;
    entityType = EntityType.Workflow;
    dataGraph: string;
    dataEditor: string;

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _projectService: ProjectService,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            if (this.vcsProject?.name === p['vcsName'] && this.repository?.name === p['repoName'] && this.workflow?.name === p['workflowName']) {
                return;
            }
            this.currentBranch = this._routeActivated?.queryParams['branch'];
            this.currentWorkflowName = p['workflowName'];
            this.loading = true;
            this._cd.markForCheck();
            forkJoin([
                this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName']),
                this._projectService.getJSONSchema(EntityType.Workflow),
                this._projectService.getJSONSchema(EntityType.Job)
            ]).subscribe(result => {
                this.repository = result[0];
                this.vcsProject = result[1];
                this.workflowJsonSchema = result[2];
                this.jobSchema = result[3];
                this._cd.markForCheck();
                this.loadWorkflow(p['workflowName'], this._routeActivated?.snapshot?.queryParams['branch']);
            });
        });
        this._routeActivated.queryParams.subscribe(q => {
            if (this.currentBranch === q['branch']) {
                return;
            }
            if (this.repository && this.vcsProject) {
                this.loadWorkflow(this.currentWorkflowName, q['branch']);
            }
            this.currentBranch = q['branch'];
            this._cd.markForCheck();
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    loadWorkflow(workflowName: string, branch?: string): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.getRepoEntity(this.project.key, this.vcsProject.name, this.repository.name, EntityType.Workflow, workflowName, branch)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wk => {
                this.errorNotFound = false;
                this.workflow = wk;
                this.dataGraph = this.workflow.data;
                this.dataEditor = this.workflow.data;
                this._cd.markForCheck();
            }, e => {
                if (e?.status === 404) {
                    delete this.workflow;
                    this.errorNotFound = true;
                    this._cd.markForCheck();
                }
            });
    }
}