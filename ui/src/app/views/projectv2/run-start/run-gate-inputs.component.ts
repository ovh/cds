import { ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, inject, Input, OnChanges, SimpleChanges } from "@angular/core";
import { V2Job, V2JobGate, V2JobGateInput } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from "@angular/forms";
import { OnChangeType, OnTouchedType } from "ng-zorro-antd/core/types";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";

export class GateValue {
    global: { [inputName: string]: any }
    withJobOverrides: boolean;
    jobs: {
        [jobName: string]: { [inputName: string]: any }
    }

    constructor() {
        this.withJobOverrides = false;
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

    disabled: boolean = false;
    values: { [gateName: string]: GateValue } = {};

    onChange: OnChangeType = () => { };
    onTouched: OnTouchedType = () => { };

    _cd = inject(ChangeDetectorRef);

    constructor() { }

    writeValue(obj: any): void { }

    registerOnChange(fn: OnChangeType): void {
        this.onChange = fn;
    }

    registerOnTouched(fn: OnTouchedType): void {
        this.onTouched = fn;
    }

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

        Object.keys(this.gates).forEach(k => {
            this.values[k] = new GateValue();
            Object.keys(this.gates[k].inputs ?? {}).forEach(v => {
                if (this.gates[k].inputs[v].default === false ) {
                    this.values[k].global[v] = false;
                } else {
                    this.values[k].global[v] = this.gates[k].inputs[v].default || undefined;
                }
            });
        });

        Object.keys(this.jobs).forEach(j => {
            if (!this.jobs[j].gate) {
                return;
            }
            this.values[this.jobs[j].gate].jobs[j] = { ...this.values[this.jobs[j].gate].global };
        });

        Object.keys(this.values).forEach(gateName => {
            // If only one job uses this gate, enable job overrides by default
            if (Object.keys(this.values[gateName].jobs).length === 1) {
                this.values[gateName].withJobOverrides = true;
            }
        });

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
            Object.keys(this.jobs).forEach(j => {
                if (!this.values[this.jobs[j].gate].withJobOverrides) {
                    result[j] = this.values[this.jobs[j].gate].global;
                } else {
                    result[j] = this.values[this.jobs[j].gate].jobs[j];
                }
            })
        }
        this.onChange(result);
    }

    asGateInput(value: any): V2JobGateInput {
        return value as V2JobGateInput;
    }
}
