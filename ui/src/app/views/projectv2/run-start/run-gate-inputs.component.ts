import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from "@angular/core";
import { V2Job, V2JobGate, V2JobGateInput } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";

@Component({
    selector: 'app-run-gate-inputs',
    templateUrl: './run-gate-inputs.html',
    styleUrls: ['./run-gate-inputs.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RunGateInputsComponent implements OnChanges {
    @Input() jobs: { [jobName: string]: V2Job } = {};
    @Input() gates: { [gateName: string]: V2JobGate } = {};
    @Output() inputsChange = new EventEmitter<{ [jobName: string]: { [inputName: string]: any } }>();

    spliJobs: boolean = false;

    // Data for non split job form
    gateNames: Array<string>;
    jobsInGates: { [gateName: string]: Array<string> };
    gateValues: { [gateName: string]: { [inputName: string]: any } } = {}; // Data use for the form

    // Data for split job form
    jobNames: Array<string>;
    jobInputsValues: { [jobName: string]: { [inputName: string]: any } } = {}; // Data use for the form

    // Input by gates
    inputs : {[gateName: string]: Array<any> }

   
    
    constructor(private _cd: ChangeDetectorRef) { }

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
        this.emitChange();
        this._cd.markForCheck();
        
    }

    onModeChange(): void {
        this.emitChange();
    }

    onValueChange(): void {
        this.emitChange();
    }

    emitChange(): void {
        const result: { [jobName: string]: { [inputName: string]: any } } = {};
        if (this.jobs) {
            Object.keys(this.jobs).forEach(j => {
                if (!this.spliJobs) {
                    result[j] = this.gateValues[this.jobs[j].gate]
                } else {
                    result[j] = this.jobInputsValues[j];
                }
            })
        }
        this.inputsChange.emit(result);
    }
}
