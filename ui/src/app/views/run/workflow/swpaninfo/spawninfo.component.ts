import {Component, Input} from '@angular/core';
import {SpawnInfo} from '../../../../model/pipeline.model';

declare var ansi_up: any;

@Component({
    selector: 'app-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss']
})
export class SpawnInfoComponent {

    @Input() spawnInfos: Array<SpawnInfo>;

    show = true;

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
        return ansi_up.ansi_to_html(msg);
    }
}
