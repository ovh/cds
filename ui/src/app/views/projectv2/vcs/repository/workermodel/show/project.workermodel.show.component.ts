import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Project, ProjectRepository, VCSProject } from 'app/model/project.model';
import { ProjectState } from 'app/store/project.state';
import { Store } from '@ngxs/store';
import { forkJoin } from 'rxjs';
import { ActivatedRoute } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { SidebarEvent, SidebarService } from 'app/service/sidebar/sidebar.service';
import { finalize } from "rxjs/operators";
import { Schema } from 'app/model/json-schema.model';
import {Entity, EntityWorkerModel} from "../../../../../../model/entity.model";

@Component({
    selector: 'app-projectv2-workermodel-show',
    templateUrl: './project.workermodel.show.html',
    styleUrls: ['./project.workermodel.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkerModelShowComponent implements OnDestroy {

    loading: boolean;
    project: Project;
    vcsProject: VCSProject;
    repository: ProjectRepository;
    workerModel: Entity;
    jsonSchema: Schema;
    currentWorkerModelName: string;
    currentBranch: string;
    errorNotFound: boolean;
    entityType = EntityWorkerModel;

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _projectService: ProjectService,
        private _cd: ChangeDetectorRef,
        private _sidebarService: SidebarService
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            if (this.vcsProject?.name === p['vcsName'] && this.repository?.name === p['repoName'] && this.workerModel?.name === p['workerModelName']) {
                return;
            }
            this.currentBranch = this._routeActivated?.queryParams['branch'];
            this.currentWorkerModelName = p['workerModelName'];
            this.loading = true;
            this._cd.markForCheck();
            forkJoin([
                this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName']),
                this._projectService.getJSONSchema(EntityWorkerModel)
            ]).subscribe(result => {
                this.repository = result[0];
                this.vcsProject = result[1];
                this.jsonSchema = result[2];
                this._cd.markForCheck();
                this.loadWorkerModel(p['workerModelName'], this._routeActivated?.snapshot?.queryParams['branch']);
            });
        });
        this._routeActivated.queryParams.subscribe(q => {
            if (this.currentBranch === q['branch']) {
                return;
            }
            if (this.repository && this.vcsProject) {
                this.loadWorkerModel(this.currentWorkerModelName, q['branch']);
            }
            this.currentBranch = q['branch'];
            this._cd.markForCheck();
        });
    }

    loadWorkerModel(workerModelName: string, branch?: string): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.getRepoEntity(this.project.key, this.vcsProject.name, this.repository.name, EntityWorkerModel, workerModelName, branch)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wm => {
                this.errorNotFound = false;
                this.workerModel = wm;
                let selectEvent = new SidebarEvent(this.workerModel.id, this.workerModel.name, EntityWorkerModel, 'select', [this.vcsProject.id, this.repository.id]);
                this._sidebarService.sendEvent(selectEvent);
                this._cd.markForCheck();
            }, e => {
                if (e?.status === 404) {
                    let selectEvent = new SidebarEvent(this.repository.id, this.repository.name, 'repository', 'select', [this.vcsProject.id]);
                    delete this.workerModel;
                    this._sidebarService.sendEvent(selectEvent);
                    this.errorNotFound = true;
                    this._cd.markForCheck();
                }
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT
}
