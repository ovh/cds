import { ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input, OnChanges, SimpleChanges } from "@angular/core";
import { V2Job, V2JobGate } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from "@angular/forms";
import { OnChangeType, OnTouchedType } from "ng-zorro-antd/core/types";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";

@Component({
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
    
    spliJobs: boolean = false;

    // Data for non split job form
    gateNames: Array<string>;
    jobsInGates: { [gateName: string]: Array<string> };
    gateValues: { [gateName: string]: { [inputName: string]: any } } = {}; // Data use for the form

    // Data for split job form
    jobNames: Array<string>;
    jobInputsValues: { [jobName: string]: { [inputName: string]: any } } = {}; // Data use for the form

    // Input by gates
    inputs : {[gateName: string]: Array<any> };

    onChange: OnChangeType = () => { };
    onTouched: OnTouchedType = () => { };

    constructor(private _cd: ChangeDetectorRef) { }

    writeValue(obj: any): void {}

    registerOnChange(fn: OnChangeType): void {
        this.onChange = fn;
    }

    registerOnTouched(fn: OnTouchedType): void {
        this.onTouched = fn;
    }

    setDisabledState?(isDisabled: boolean): void {}

    ngOnChanges(changes: SimpleChanges): void {
        if (changes.jobs || changes.gates) {
            this.init();
        }
    }

    init(): void {
        this.gateNames = new Array<string>();
        this.inputs = {};
        this.gateValues = {};
        this.jobsInGates = {};
        this.jobNames = [];
        this.jobInputsValues = {};
        if (this.gates) {
            Object.keys(this.gates).forEach( k => {
                this.gateNames.push(k);
                this.gateValues[k] = {};
                this.inputs[k] = new Array<any>();
                this.jobsInGates[k] = [];
                if (this.gates[k].inputs) {
                    Object.keys(this.gates[k].inputs).forEach(v => {
                        this.inputs[k].push({"name": v, "data": this.gates[k].inputs[v]});
                        this.gateValues[k][v] = undefined;
                    });
                }
            });
        }
        if (this.jobs) {
            Object.keys(this.jobs).forEach(j => {
                this.jobsInGates[this.jobs[j].gate].push(j)
                this.jobNames.push(j);
                this.jobInputsValues[j] = {};

                // Init data use by the form
                Object.keys(this.gates[this.jobs[j].gate].inputs).forEach(v => {
                    this.jobInputsValues[j][v] = undefined;
                })
                
            })
        }
        this._cd.markForCheck();
    }

    onModeChange(event : boolean): void {
        this.spliJobs = event;
        this._cd.markForCheck();
        this.emitChange();
    }

    onGateValueChange(gate: string, input: string, event: any): void {
        this.gateValues[gate][input] = event;
        this.emitChange();
    }

    onJobValueChange(jobName: string, input: string, event: any): void {
        this.jobInputsValues[jobName][input] = event;
        this.emitChange();
    }

    emitChange(): void {
        let result = {};
        if (this.jobs) {
            Object.keys(this.jobs).forEach(j => {
                if (!this.spliJobs) {
                    result[j] = this.gateValues[this.jobs[j].gate]
                } else {
                    result[j] = this.jobInputsValues[j];
                }
            })
        }
        this.onChange(result);
    }
}
