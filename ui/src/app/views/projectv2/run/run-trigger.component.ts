import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnInit } from "@angular/core";
import { FormBuilder, FormControl, FormGroup } from "@angular/forms";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { areAllJobVariantsSelected, V2Job, V2JobGate, V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunTriggerJobsRequest } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { NzDrawerRef } from "ng-zorro-antd/drawer";
import { NzMessageService } from "ng-zorro-antd/message";
import { ErrorUtils } from "app/shared/error.utils";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { lastValueFrom } from "rxjs";

@Component({
    standalone: false,
    selector: 'app-run-trigger',
    templateUrl: './run-trigger.html',
    styleUrls: ['./run-trigger.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunTriggerComponent implements OnInit {
    @Input() run: V2WorkflowRun;
    @Input() runJobs: Array<V2WorkflowRunJob>;
    @Input() jobRunIDs: Array<string>;

    validateForm: FormGroup<{
        jobInputs: FormControl<{ [jobIdentifier: string]: { [inputName: string]: any } } | null>
    }>;
    initialValues: { [jobIdentifier: string]: { [inputName: string]: any } } = {};
    allJobLabels: Array<string> = [];
    gatedJobLabels: Array<string> = [];
    loading: boolean = false;
    error: string;

    // Variables for app-gate-inputs component
    gatedJobs: { [jobID: string]: V2Job } = {};
    gates: { [gateName: string]: V2JobGate } = {};

    private _cd = inject(ChangeDetectorRef);
    private _fb = inject(FormBuilder);
    private _drawerRef = inject<NzDrawerRef<boolean>>(NzDrawerRef);
    private _workflowService = inject(V2WorkflowRunService);
    private _messageService = inject(NzMessageService);

    constructor() {
        this.validateForm = this._fb.group({
            jobInputs: this._fb.control<{ [jobIdentifier: string]: { [inputName: string]: any } } | null>(null)
        });
    }

    ngOnInit(): void {
        this.init();
    }

    init(): void {
        this.gatedJobs = {};
        this.gates = {};
        this.initialValues = {};

        // Determine all job labels and gated job labels, and initialize gate inputs values
        for (const jobRunID of this.jobRunIDs) {
            const runJob = this.runJobs.find(j => j.id === jobRunID);
            const isPartialMatrixSelection = runJob.matrix && !areAllJobVariantsSelected(runJob.job_id, this.jobRunIDs, this.runJobs);
            if (isPartialMatrixSelection) {
                const matrixLabel = Object.entries(runJob.matrix).map(([k, v]) => `${k}:${v}`).join(', ');
                this.allJobLabels.push(`${runJob.job_id} (${matrixLabel})`);
                continue;
            }
            if (this.allJobLabels.find(j => j === runJob.job_id)) {
                continue;
            }

            // For gated jobs, prepare data for gate inputs form
            if (runJob.job.gate) {
                const gate = this.run.workflow_data.workflow.gates[runJob.job.gate]
                if (!gate.inputs || Object.keys(gate.inputs).length === 0) {
                    this.allJobLabels.push(runJob.job_id);
                    continue;
                }
                this.gatedJobLabels.push(runJob.job_id);
                this.gatedJobs[runJob.job_id] = runJob.job;
                this.gates[runJob.job.gate] = gate;

                // Only populate initialValues if gate_inputs actually exists (gate was triggered before).
                // An empty object {} is truthy and would mask the gate default values in RunGateInputsComponent.
                if (runJob.gate_inputs) {
                    this.initialValues[runJob.job_id] = {};
                    for (const inputName in gate.inputs || {}) {
                        this.initialValues[runJob.job_id][inputName] = runJob.gate_inputs[inputName];
                    }
                }
            }

            this.allJobLabels.push(runJob.job_id);
        }

        this._cd.markForCheck();
    }

    async submitForm() {
        if (!this.validateForm.valid) {
            Object.values(this.validateForm.controls).forEach(control => {
                if (control.invalid) {
                    control.markAsDirty();
                    control.updateValueAndValidity({ onlySelf: true });
                }
            });
            return;
        }

        this.loading = true;
        this.error = null;
        this.validateForm.disable();
        this._cd.markForCheck();

        let payload = { job_inputs: {} };
        for (const jobRunID of this.jobRunIDs) {
            const runJob = this.runJobs.find(j => j.id === jobRunID);
            const isPartialMatrixSelection = runJob.matrix && !areAllJobVariantsSelected(runJob.job_id, this.jobRunIDs, this.runJobs);
            if (isPartialMatrixSelection) {
                payload.job_inputs[jobRunID] = {};
                continue;
            }
            if (payload.job_inputs[runJob.job_id]) {
                continue;
            }
            payload.job_inputs[runJob.job_id] = this.validateForm.value.jobInputs?.[runJob.job_id] || {};
        }

        try {
            await lastValueFrom(this._workflowService.triggerJobs(
                this.run.project_key, this.run.id,
                payload
            ));
            const count = Object.keys(payload.job_inputs).length;
            this._messageService.success(
                `${count} job${count > 1 ? 's' : ''} restarted successfully`,
                { nzDuration: 2000 }
            );
            this._drawerRef.close(true);
        } catch (e) {
            this.error = ErrorUtils.print(e);
            return;
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }
    }

    clearError(): void {
        this.error = null;
        this.validateForm.enable();
        this._cd.markForCheck();
    }
}
