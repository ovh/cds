import {Component, Input, OnInit} from '@angular/core';
import {VCSConnections, VCSStrategy} from '../../model/vcs.model';
import {Project} from '../../model/project.model';
import {KeyService} from '../../service/keys/keys.service';

@Component({
    selector: 'app-vcs-strategy',
    templateUrl: './vcs.strategy.html',
    styleUrls: ['./vcs.strategy.scss']
})
export class VCSStrategyComponent implements OnInit {

    @Input() project: Project;

    _strategy: VCSStrategy;
    @Input('strategy')
    set strategy(data: VCSStrategy) {
        if (data) {
            this._strategy = data;
        }
    }
    get strategy() {
        return this._strategy;
    }

    keys: Array<string>;
    connectionType = VCSConnections;

    constructor(private _keyService: KeyService) { }

    ngOnInit() {
        if (!this.strategy) {
            this.strategy = new VCSStrategy();
        }
        this._keyService.getAllKeys(this.project.key).subscribe(k => {
            this.keys = k;
        })
    }
}
