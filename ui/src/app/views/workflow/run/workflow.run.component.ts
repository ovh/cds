import {Component, NgZone} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {CDSWorker} from '../../../shared/worker/worker';
import {WorkflowRun} from '../../../model/workflow.run.model';
import {environment} from '../../../../environments/environment';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {WorkflowJoinComponent} from '../../../shared/workflow/join/workflow.join.component';
import {WorkflowNodeComponent} from '../../../shared/workflow/node/workflow.node.component';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss']
})
export class WorkflowRunComponent {

    project: Project;
    runWorkflowWorker: CDSWorker;
    workflowRun: WorkflowRun;
    zone: NgZone;

    selectedTab = '';

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore, private _router: Router) {
        this.zone = new NgZone({enableLongStackTrace: false});
        // Update data if route change
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });
        this._activatedRoute.queryParams.subscribe(q => {
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'workflow';
            }
        });

        this._activatedRoute.params.first().subscribe(params => {
            let key = params['key'];
            let workflowName = params['workflowName'];
            let number = params['number'];
            if (key && workflowName && number) {
                // Start web worker
                this.runWorkflowWorker = new CDSWorker('./assets/worker/web/workflow2.js');
                this.runWorkflowWorker.start({
                    'user': this._authStore.getUser(),
                    'session': this._authStore.getSessionToken(),
                    'api': environment.apiURL,
                    key: key,
                    workflowName: workflowName,
                    number: number
                });
                this.runWorkflowWorker.response().subscribe(wrString => {
                    this.zone.run(() => {
                       this.workflowRun = <WorkflowRun>JSON.parse(wrString);
                    });
                });
            }
        });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key +
            '/workflow/' + this.workflowRun.workflow.name +
            '/run/' + this.workflowRun.num +
            '?&tab=' + tab);
    }
}
