import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/internal/operators/finalize';
import { Action } from '../../../../model/action.model';
import { Group } from '../../../../model/group.model';
import { Job } from '../../../../model/job.model';
import { Pipeline } from '../../../../model/pipeline.model';
import { Stage } from '../../../../model/stage.model';
import { ActionService } from '../../../../service/action/action.service';
import { GroupService } from '../../../../service/group/group.service';
import { PipelineService } from '../../../../service/pipeline/pipeline.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-action-add',
    templateUrl: './action.add.html',
    styleUrls: ['./action.add.scss']
})
@AutoUnsubscribe()
export class ActionAddComponent implements OnInit {
    action: Action;
    groups: Array<Group>;
    loading: boolean;
    path: Array<PathItem>;
    queryParamsSub: Subscription;
    projectKey: string;
    pipeline: Pipeline;
    stage: Stage;
    job: Job;

    constructor(
        private _actionService: ActionService,
        private _pipelineService: PipelineService,
        private _toast: ToastService, private _translate: TranslateService,
        private _router: Router,
        private _groupService: GroupService,
        private _route: ActivatedRoute
    ) {
        this.action = <Action>{ editable: true };
        this.getGroups();

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'action_list_title',
            routerLink: ['/', 'settings', 'action']
        }, <PathItem>{
            translate: 'common_create'
        }];
    }

    ngOnInit() {
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['from']) {
                let path = params['from'].split('/');
                if (path.length === 4) {
                    this.initFromJob(path[0], path[1], Number(path[2]), path[3]);
                }
            }
        });
    }

    initFromJob(projectKey: string, pipelineName: string, stageID: number, jobName: string) {
        this.projectKey = projectKey;
        this._pipelineService.getPipeline(projectKey, pipelineName).subscribe(p => {
            this.pipeline = p;
            this.stage = this.pipeline.stages.find((s: Stage) => { return s.id === stageID; });
            if (this.stage) {
                this.job = this.stage.jobs.find((j: Job) => { return j.action.name === jobName; });
                if (this.job) {
                    this.action = <Action>{
                        editable: true,
                        description: this.job.action.description,
                        requirements: this.job.action.requirements,
                        actions: this.job.action.actions
                    };
                }
            }
        });
    }

    getGroups() {
        this.loading = true;
        this._groupService.getGroups()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    actionSave(action: Action): void {
        this.action.loading = true;
        this._actionService.add(action).subscribe(a => {
            this._toast.success('', this._translate.instant('action_saved'));
            // navigate to have action name in url
            this._router.navigate(['settings', 'action', a.group.name, a.name]);
        }, () => {
            this.action.loading = false;
        });
    }
}
