import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit, Output } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { ToastService } from "app/shared/toast/ToastService";
import { V2JobGate, V2WorkflowRun } from "../../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { lastValueFrom } from "rxjs";
import { NzMessageService } from "ng-zorro-antd/message";
import { ErrorUtils } from "app/shared/error.utils";

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
    @Output() onSubmit = new EventEmitter<void>();

    currentGate: V2JobGate;
    request: { [key: string]: any };
    loading: boolean;

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowService: V2WorkflowRunService,
        private _toastService: ToastService,
        private _messageService: NzMessageService
    ) { }

    ngOnInit(): void {
        this.currentGate = <V2JobGate>this.run.workflow_data.workflow.gates[this.gateNode.gate];
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
            const jobEvent = this.run.job_events.find(je => je.job_id === this.gateNode.job && je.run_attempt === this.run.run_attempt);
            if (jobEvent) {
                Object.keys(jobEvent.inputs).forEach(k => {
                    this.request[k] = jobEvent.inputs[k];
                });
            }
        }
        this._cd.markForCheck();
    }

    async triggerJob() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._workflowService.triggerJob(this.run.project_key, this.run.id, this.gateNode.job, this.request));
            this._toastService.success('', `Job ${this.gateNode.job} started`);
            this.onSubmit.emit();
        } catch (e) {
            this._messageService.error(`Unable to get trigger job gate: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }
}
