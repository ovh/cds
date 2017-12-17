import {Component, OnInit} from '@angular/core';
import {MonitoringService} from '../../../service/monitoring/monitoring.service';
import { forEach } from '@angular/router/src/utils/collection';
import { MonitoringStatus } from 'app/model/monitoring.model';

@Component({
    selector: 'app-status',
    templateUrl: './status.html',
    styleUrls: ['./status.scss']
})
export class StatusComponent implements OnInit {

    status: MonitoringStatus;

    constructor(private _monitoringService: MonitoringService) {
        this._monitoringService.getStatus()
            .subscribe(r => {
                this.status = r;
            });
    }

    ngOnInit() {
    }
}
