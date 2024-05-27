import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import {
    Project,
    ProjectRepository
} from "app/model/project.model";
import { Schema } from "app/model/json-schema.model";
import { Store } from "@ngxs/store";
import { ActivatedRoute } from "@angular/router";
import { ProjectService } from "app/service/project/project.service";
import { ProjectState } from "app/store/project.state";
import { forkJoin } from "rxjs";
import { finalize } from "rxjs/operators";
import { Entity, EntityType } from "app/model/entity.model";
import { VCSProject } from "app/model/vcs.model";

@Component({
    selector: 'app-projectv2-action-show',
    templateUrl: './project.action.show.html',
    styleUrls: ['./project.action.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ActionShowComponent implements OnDestroy {

    loading: boolean;
    project: Project;
    vcsProject: VCSProject;
    repository: ProjectRepository;
    action: Entity;
    jsonSchema: Schema;
    currentActionName: string;
    currentBranch: string;
    errorNotFound: boolean;
    entityType = EntityType.Action;

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _projectService: ProjectService,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            if (this.vcsProject?.name === p['vcsName'] && this.repository?.name === p['repoName'] && this.action?.name === p['actionName']) {
                return;
            }
            this.currentBranch = this._routeActivated?.queryParams['branch'];
            this.currentActionName = p['actionName'];
            this.loading = true;
            this._cd.markForCheck();
            forkJoin([
                this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName']),
                this._projectService.getJSONSchema(EntityType.Action)
            ]).subscribe(result => {
                this.repository = result[0];
                this.vcsProject = result[1];
                this.jsonSchema = result[2];
                this._cd.markForCheck();
                this.loadAction(p['actionName'], this._routeActivated?.snapshot?.queryParams['branch']);
            });
        });
        this._routeActivated.queryParams.subscribe(q => {
            if (this.currentBranch === q['branch']) {
                return;
            }
            if (this.repository && this.vcsProject) {
                this.loadAction(this.currentActionName, q['branch']);
            }
            this.currentBranch = q['branch'];
            this._cd.markForCheck();
        });
    }

    loadAction(actionName: string, branch?: string): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.getRepoEntity(this.project.key, this.vcsProject.name, this.repository.name, EntityType.Action, actionName, branch)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(act => {
                this.errorNotFound = false;
                this.action = act;
                this._cd.markForCheck();
            }, e => {
                if (e?.status === 404) {
                    delete this.action;
                    this.errorNotFound = true;
                    this._cd.markForCheck();
                }
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

}

