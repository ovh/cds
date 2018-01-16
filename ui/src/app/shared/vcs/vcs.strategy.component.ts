import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {VCSConnections, VCSStrategy} from '../../model/vcs.model';
import {Project} from '../../model/project.model';
import {KeyService} from '../../service/keys/keys.service';
import {AllKeys} from '../../model/keys.model';

@Component({
    selector: 'app-vcs-strategy',
    templateUrl: './vcs.strategy.html',
    styleUrls: ['./vcs.strategy.scss']
})
export class VCSStrategyComponent implements OnInit {

    @Input() project: Project;
    @Input() appName: string;

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

    @Output() strategyChange = new EventEmitter<VCSStrategy>();
    keys: AllKeys;
    connectionType = VCSConnections;
    displayVCSStrategy = false;


    constructor(private _keyService: KeyService) { }

    ngOnInit() {
        if (!this.strategy) {
            this.strategy = new VCSStrategy();
        }
        this._keyService.getAllKeys(this.project.key, this.appName).subscribe(k => {
            this.keys = k;
        })
    }

    saveStrategy() {
        this.strategyChange.emit(this.strategy);
    }
}
