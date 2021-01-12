import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
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
    styleUrls: ['./action.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ActionAddComponent implements OnInit, OnDestroy {
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
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef
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

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['from']) {
                let path = params['from'].split('/');
                if (path.length === 4) {
                    this.initFromJob(path[0], path[1], Number(path[2]), path[3]);
                }
                this._cd.markForCheck();
            }
        });
    }

    initFromJob(projectKey: string, pipelineName: string, stageID: number, jobName: string) {
        this.projectKey = projectKey;
        this._pipelineService.getPipeline(projectKey, pipelineName)
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(p => {
            this.pipeline = p;
            this.stage = this.pipeline.stages.find((s: Stage) => s.id === stageID);
            if (this.stage) {
                this.job = this.stage.jobs.find((j: Job) => j.action.name === jobName);
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
        this._groupService.getAll()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    actionSave(action: Action): void {
        this.action.loading = true;
        this._actionService.add(action)
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(a => {
            this._toast.success('', this._translate.instant('action_saved'));
            // navigate to have action name in url
            this._router.navigate(['settings', 'action', a.group.name, a.name]);
        }, () => {
            this.action.loading = false;
        });
    }
}
