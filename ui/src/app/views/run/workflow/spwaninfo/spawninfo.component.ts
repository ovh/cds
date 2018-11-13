import {Component, Input, ViewChild} from '@angular/core';
import * as AU from 'ansi_up';
import {Parameter} from '../../../../model/parameter.model';
import {SpawnInfo} from '../../../../model/pipeline.model';
import {JobVariableComponent} from '../variables/job.variables.component';

@Component({
    selector: 'app-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss']
})
export class SpawnInfoComponent {

    @Input() spawnInfos: Array<SpawnInfo>;
    @Input() variables: Array<Parameter>;

    @ViewChild('jobVariable')
    jobVariable: JobVariableComponent;

    show = true;
    ansi_up = new AU.default;

    constructor() { }

    toggle() {
        this.show = ! this.show;
    }

    getSpawnInfos() {
        let msg = '';
        if (this.spawnInfos) {
            this.spawnInfos.forEach( s => {
               msg += '[' + s.api_time.toString().substr(0, 19) + '] ' + s.user_message + '\n';
            });
        }
        if (msg !== '') {
            return this.ansi_up.ansi_to_html(msg);
        }
        return '';
    }

    openVariableModal(): void {
        if (this.jobVariable) {
            this.jobVariable.show({autofocus: false, closable: false, observeChanges: true});
        }
    }
}
