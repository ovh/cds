import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnInit } from "@angular/core";
import { FormBuilder, FormControl, FormGroup } from "@angular/forms";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { V2Job, V2JobGate, V2WorkflowRun, V2WorkflowRunJob } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
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
    @Input() jobs: { [jobName: string]: V2Job };
    @Input() gates: { [gateName: string]: V2JobGate };
    // Non-gated and matrix job inputs to include in a restart batch call.
    // When set, the component uses startJobs API; otherwise triggerJob API.
    @Input() additionalJobInputs: { [jobId: string]: { [inputName: string]: any } };
    @Input() runJobs: Array<V2WorkflowRunJob>;

    validateForm: FormGroup<{
        jobInputs: FormControl<{ [jobName: string]: { [inputName: string]: any } } | null>
    }>;
    initialValues: { [jobName: string]: { [inputName: string]: any } } = {};
    allJobEntries: { id: string, label: string }[];
    gatedJobNames: string[];
    loading: boolean;
    error: string;

    private _cd = inject(ChangeDetectorRef);
    private _fb = inject(FormBuilder);
    private _drawerRef = inject<NzDrawerRef<boolean>>(NzDrawerRef);
    private _workflowService = inject(V2WorkflowRunService);
    private _messageService = inject(NzMessageService);

    ngOnInit(): void {
        this.validateForm = this._fb.group({
            jobInputs: this._fb.control<{ [jobName: string]: { [inputName: string]: any } } | null>(null),
        });
        this.init();
    }

    init(): void {
        this.loading = false;
        this.error = null;

        this.allJobEntries = [
            ...Object.keys(this.jobs).map(id => ({ id, label: id })),
            ...Object.keys(this.additionalJobInputs ?? {}).map(id => ({
                id,
                label: this.buildJobLabel(id)
            }))
        ];

        this.gatedJobNames = Object.keys(this.jobs).filter(id => {
            const job = this.jobs[id];
            return job.gate && this.gates[job.gate];
        });

        // Compute initial values from previous job_events
        this.initialValues = {};
        if (this.run.job_events) {
            Object.keys(this.jobs).forEach(jobName => {
                const jobEvent = this.run.job_events.find(
                    je => je.job_id === jobName && je.run_attempt === this.run.run_attempt
                );
                if (jobEvent && jobEvent.inputs) {
                    this.initialValues[jobName] = { ...jobEvent.inputs };
                }
            });
        }

        this._cd.markForCheck();
    }

    async submitForm() {
        this.loading = true;
        this.error = null;
        this.validateForm.disable();
        this._cd.markForCheck();

        try {
            if (this.additionalJobInputs) {
                await this.startJobs();
            } else {
                await this.triggerSingleJob();
            }
            this._drawerRef.close(true);
        } catch (e) {
            this.error = ErrorUtils.print(e);
            this.loading = false;
            this._cd.markForCheck();
            return;
        }

        this.validateForm.enable();
        this.loading = false;
        this._cd.markForCheck();
    }

    clearError(): void {
        this.error = null;
        this.validateForm.enable();
        this._cd.markForCheck();
    }

    async triggerSingleJob(): Promise<void> {
        const jobId = Object.keys(this.jobs)[0];
        const inputs = this.validateForm.value.jobInputs?.[jobId] || {};
        await lastValueFrom(this._workflowService.triggerJob(
            this.run.project_key, this.run.id, jobId, inputs
        ));
        this._messageService.success(`Job ${jobId} started`);
    }

    async startJobs(): Promise<void> {
        const jobInputs: { [id: string]: { [inputName: string]: any } } = {};
        const formInputs = this.validateForm.value.jobInputs ?? {};

        for (const jobId of Object.keys(this.jobs)) {
            jobInputs[jobId] = formInputs[jobId] ?? {};
        }
        for (const [id, inputs] of Object.entries(this.additionalJobInputs)) {
            jobInputs[id] = inputs;
        }

        await lastValueFrom(this._workflowService.startJobs(
            this.run.project_key, this.run.id,
            { job_inputs: jobInputs }
        ));

        const count = Object.keys(jobInputs).length;
        this._messageService.success(
            `${count} job${count > 1 ? 's' : ''} restarted successfully`,
            { nzDuration: 2000 }
        );
    }

    buildJobLabel(id: string): string {
        const runJob = this.runJobs?.find(j => j.id === id);
        if (runJob?.matrix) {
            const matrixLabel = Object.entries(runJob.matrix).map(([k, v]) => `${k}:${v}`).join(', ');
            return `${runJob.job_id} (${matrixLabel})`;
        }
        return id;
    }
}
