import {ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {Entity, EntityWorkerModel, Project, ProjectRepository, VCSProject} from 'app/model/project.model';
import { ProjectState } from 'app/store/project.state';
import { Store } from '@ngxs/store';
import { forkJoin } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import {SidebarEvent, SidebarService} from 'app/service/sidebar/sidebar.service';
import {FlatNodeItem} from "../../../../../../shared/tree/tree.component";

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
    jsonSchema: any;

    constructor(private _store: Store, private _routeActivated: ActivatedRoute, private _projectService: ProjectService,
                private _cd: ChangeDetectorRef, private _toastService: ToastService, private _router: Router, private _sidebarService: SidebarService) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            if (this.vcsProject?.name === p['vcsName'] && this.repository?.name === p['repoName'] && this.workerModel?.name === p['workerModelName']) {
                return;
            }

            forkJoin( [
                this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName']),
                this._projectService.getRepoEntity(this.project.key, p['vcsName'], p['repoName'], EntityWorkerModel, p['workerModelName']),
                this._projectService.getJSONSchema(EntityWorkerModel)
            ]).subscribe(result => {
                this.repository = result[0];
                this.vcsProject = result[1];
                this.workerModel = result[2]
                this.jsonSchema = result[3];
                let selectEvent = new SidebarEvent(this.workerModel.id, this.workerModel.name, EntityWorkerModel, 'select', [this.vcsProject.id, this.repository.id]);
                this._sidebarService.sendEvent(selectEvent);
                this._cd.markForCheck();
            });


        });
    }

    ngOnDestroy() {}
}
