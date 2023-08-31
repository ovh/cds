import {ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy} from "@angular/core";
import {AutoUnsubscribe} from "../../../shared/decorator/autoUnsubscribe";
import {SidebarService} from "../../../service/sidebar/sidebar.service";
import {Subscription} from "rxjs";
import {V2WorkflowRun, V2WorkflowRunJob} from "../../../model/v2.workflow.run.model";
import {dump, load, LoadOptions} from "js-yaml";
import {V2WorkflowRunService} from "../../../service/workflowv2/workflow.service";
import {first} from "rxjs/operators";


@Component({
    selector: 'app-projectv2-run',
    templateUrl: './project.run.html',
    styleUrls: ['./project.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowRunComponent implements OnDestroy {

    selectedRun: V2WorkflowRun;
    jobs: Array<V2WorkflowRunJob>;
    workflowGraph: any;
    sidebarSubs: Subscription;

    constructor(private _sidebarService: SidebarService, private _cd: ChangeDetectorRef, private _workflowService: V2WorkflowRunService) {
        this.sidebarSubs = this._sidebarService.getRunObservable().subscribe(r => {
            this.selectedRun = r;
            if (r) {
                this.workflowGraph = dump(r.workflow_data.workflow);
                this._workflowService.getJobs(r).pipe(first()).subscribe(jobs => {
                    this.jobs = jobs;
                    this._cd.markForCheck();
                });
            } else {
                delete this.workflowGraph;
            }
            this._cd.markForCheck();
        });
    }

    ngOnDestroy(): void {
    }

}
