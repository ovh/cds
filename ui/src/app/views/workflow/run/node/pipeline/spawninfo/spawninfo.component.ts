import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {Job} from '../../../../../../model/job.model';
import {Parameter} from '../../../../../../model/parameter.model';
import {SpawnInfo} from '../../../../../../model/pipeline.model';
import {JobVariableComponent} from '../../../../../run/workflow/variables/job.variables.component';

declare var ansi_up: any;

@Component({
    selector: 'app-workflow-rin-job-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss']
})
export class WorkflowRunJobSpawnInfoComponent implements OnInit {

    @Input() spawnInfos: Array<SpawnInfo>;
    @Input() variables: Array<Parameter>;
    @Input() job: Job;
    @Input('displayServiceLogs')
    set displayServiceLogs(data: boolean) {
        this._displayServiceLogs = data;
        this.displayServicesLogsChange.emit(data);
    }
    get displayServiceLogs(): boolean {
        return this._displayServiceLogs;
    }

    @Output() displayServicesLogsChange = new EventEmitter<boolean>();

    @ViewChild('jobVariable')
    jobVariable: JobVariableComponent;

    show = true;
    displayServiceLogsLink = false;
    _displayServiceLogs: boolean;

    constructor() { }

    ngOnInit() {
        if (this.job && this.job.action && Array.isArray(this.job.action.requirements)) {
            this.displayServiceLogsLink = this.job.action.requirements.some((req) => req.type === 'service');
        }
    }

    toggle() {
        this.show = !this.show;
    }

    getSpawnInfos() {
        let msg = '';
        if (this.spawnInfos) {
            this.spawnInfos.forEach( s => {
               msg += '[' + s.api_time.toString().substr(0, 19) + '] ' + s.user_message + '\n';
            });
        }
        if (msg !== '') {
            return ansi_up.ansi_to_html(msg);
        }
        return '';
    }

    openVariableModal(event: Event): void {
        event.stopPropagation();
        if (this.jobVariable) {
            this.jobVariable.show({autofocus: false, closable: false, observeChanges: true});
        }
    }
}
