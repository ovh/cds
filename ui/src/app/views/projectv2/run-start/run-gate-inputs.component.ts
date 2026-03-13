import { ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, inject, Input, OnChanges, SimpleChanges } from "@angular/core";
import { V2Job, V2JobGate } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from "@angular/forms";
import { OnChangeType, OnTouchedType } from "ng-zorro-antd/core/types";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";

export class GateValue {
    default: { [inputName: string]: any }
    global: { [inputName: string]: any }
    withJobOverrides: boolean;
    jobs: {
        [jobName: string]: { [inputName: string]: any }
    }

    constructor() {
        this.withJobOverrides = false;
        this.default = {};
        this.global = {};
        this.jobs = {};
    }

    getJobsNames(): string {
        return Object.keys(this.jobs).join(', ');
    }

    getJobsCount(): number {
        return Object.keys(this.jobs).length;
    }
}

@Component({
    standalone: false,
    selector: 'app-run-gate-inputs',
    templateUrl: './run-gate-inputs.html',
    styleUrls: ['./run-gate-inputs.scss'],
    providers: [
        {
            provide: NG_VALUE_ACCESSOR,
            useExisting: forwardRef(() => RunGateInputsComponent),
            multi: true
        }
    ],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunGateInputsComponent implements ControlValueAccessor, OnChanges {
    @Input() jobs: { [jobName: string]: V2Job } = {};
    @Input() gates: { [gateName: string]: V2JobGate } = {};
    @Input() initialValues: { [jobName: string]: { [inputName: string]: any } } = {};

    disabled: boolean = false;
    values: { [gateName: string]: GateValue } = {};

    onChange: OnChangeType = () => { };
    onTouched: OnTouchedType = () => { };

    _cd = inject(ChangeDetectorRef);

    constructor() { }

    writeValue(obj: any): void { }

    registerOnChange(fn: OnChangeType): void {
        this.onChange = fn;
        // Emit current value now that the form control is connected.
        // init() may have already been called from ngOnChanges before
        // registerOnChange, so the initial emitChange() was a no-op.
        this.emitChange();
    }

    registerOnTouched(fn: OnTouchedType): void { this.onTouched = fn; }

    setDisabledState?(isDisabled: boolean): void {
        this.disabled = isDisabled;
        this._cd.markForCheck();
    }

    ngOnChanges(changes: SimpleChanges): void {
        if (this.jobs && this.gates) {
            this.init();
        }
    }

    init(): void {
        this.values = {};

        // Initialize gate default values
        for (const gateName of Object.keys(this.gates)) {
            this.values[gateName] = new GateValue();
            for (const v of Object.keys(this.gates[gateName].inputs ?? {})) {
                if (this.gates[gateName].inputs[v].default === false) {
                    this.values[gateName].default[v] = false;
                } else {
                    this.values[gateName].default[v] = this.gates[gateName].inputs[v].default || undefined;
                }
            }
        }

        // Apply initial values for each job input or fallback to gate default values
        for (const j of Object.keys(this.jobs)) {
            if (!this.jobs[j].gate) {
                continue
            }
            if (this.initialValues && this.initialValues[j]) {
                this.values[this.jobs[j].gate].jobs[j] = { ...this.initialValues[j] };
            } else {
                this.values[this.jobs[j].gate].jobs[j] = { ...this.values[this.jobs[j].gate].default };
            }
        }

        // Determine if job overrides should be enabled by default based on initial values
        for (const gateName of Object.keys(this.values)) {
            // If only one job uses this gate, enable job overrides by default and init global values with the job values
            if (Object.keys(this.values[gateName].jobs).length === 1) {
                this.values[gateName].withJobOverrides = true;
                this.values[gateName].global = { ...this.values[gateName].jobs[Object.keys(this.values[gateName].jobs)[0]] };
                continue;
            }

            // For gates with multiple jobs, check if all jobs have the same initial values for each input. If so, disable job overrides by default and init global values with the common value
            let allJobsHaveSameInitialValue = true;
            let commonValues: { [inputName: string]: any } = null;
            for (const jobIdentifier of Object.keys(this.values[gateName].jobs)) {
                if (!commonValues) {
                    commonValues = { ...this.values[gateName].jobs[jobIdentifier] };
                    continue
                }
                const jobValues = this.values[gateName].jobs[jobIdentifier];
                allJobsHaveSameInitialValue = JSON.stringify(commonValues) === JSON.stringify(jobValues);
                if (!allJobsHaveSameInitialValue) {
                    break;
                }
            }
            if (allJobsHaveSameInitialValue) {
                this.values[gateName].withJobOverrides = false;
                this.values[gateName].global = { ...commonValues };
            } else {
                this.values[gateName].withJobOverrides = true;
                this.values[gateName].global = { ...this.values[gateName].default };
            }
        }

        this.emitChange();

        this._cd.markForCheck();
    }

    onGateValueChange(gate: string, input: string, event: any): void {
        this.values[gate].global[input] = event;
        this.emitChange();
    }

    onJobValueChange(gate: string, jobName: string, input: string, event: any): void {
        this.values[gate].jobs[jobName][input] = event;
        this.emitChange();
    }

    onGateWithJobOverridesChange(gate: string, event: boolean): void {
        this.values[gate].withJobOverrides = event;
        this._cd.markForCheck();
    }

    emitChange(): void {
        let result = {};
        if (this.jobs) {
            for (const j of Object.keys(this.jobs)) {
                if (!this.values[this.jobs[j].gate].withJobOverrides) {
                    result[j] = this.values[this.jobs[j].gate].global;
                } else {
                    result[j] = this.values[this.jobs[j].gate].jobs[j];
                }
            }
        }
        this.onChange(result);
    }
}
