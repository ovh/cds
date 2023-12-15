import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit} from "@angular/core";
import {AutoUnsubscribe} from "app/shared/decorator/autoUnsubscribe";
import {Gate, V2WorkflowRun} from "../../../../model/v2.workflow.run.model";

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
    request : {[key:string]: any};

    constructor(private _cd: ChangeDetectorRef) {
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
        console.log(this.request);
        this._cd.markForCheck();
    }
}