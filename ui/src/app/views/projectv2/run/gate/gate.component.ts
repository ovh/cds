import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit} from "@angular/core";
import {AutoUnsubscribe} from "app/shared/decorator/autoUnsubscribe";
import {finalize, first} from "rxjs/operators";
import {Gate, V2WorkflowRun, V2WorkflowRunJobEvent} from "app/model/v2.workflow.run.model";
import {V2WorkflowRunService} from "app/service/workflowv2/workflow.service";
import {ToastService} from "app/shared/toast/ToastService";

@Component({
    selector: 'app-run-gate',
    templateUrl: './gate.html',
    styleUrls: ['./gate.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunGateComponent implements OnInit {

    @Input() run: V2WorkflowRun;
    @Input() gateNode: { gate, job };

    currentGate: Gate;
    jobEvent: V2WorkflowRunJobEvent;
    request : {[key:string]: any};
    loading: boolean;

    constructor(private _cd: ChangeDetectorRef, private _workflowService: V2WorkflowRunService, private _toastService: ToastService) {
    }

    ngOnInit(): void {
        this.currentGate = <Gate>this.run.workflow_data.workflow.gates[this.gateNode.gate];
        this.request = {};
        Object.keys(this.currentGate.inputs).forEach(k => {
            if (this.currentGate.inputs[k].default) {
                this.request[k] = this.currentGate.inputs[k].default;
            } else {
                switch (this.currentGate.inputs[k].type) {
                    case 'boolean':
                        this.request[k] = false;
                        break;
                    case 'number':
                        this.request[k] = 0;
                        break;
                    default:
                        this.request[k] = '';
                }
            }

        });
        if (this.run.job_events) {
            this.run.job_events.forEach(je => {
                if (je.job_id === this.gateNode.job) {
                    this.jobEvent = je;
                }
            });
        }
        this._cd.markForCheck();
    }

    triggerJob(): void {
        this.loading = true;
        this._workflowService.triggerJob(this.run, this.gateNode.job, this.request)
            .pipe(first(), finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toastService.success('', `job ${this.gateNode.job} started`)
            });
        this._cd.markForCheck();
    }
}
