import {Component} from '@angular/core';
import { MonitoringStatus } from 'app/model/monitoring.model';
import {MonitoringService} from '../../../service/monitoring/monitoring.service';

@Component({
    selector: 'app-status',
    templateUrl: './status.html',
    styleUrls: ['./status.scss']
})
export class StatusComponent {

    status: MonitoringStatus;
    loading = false;

    constructor(private _monitoringService: MonitoringService) {
        this.loading = true;
        this._monitoringService.getStatus()
            .subscribe(r => {
                this.status = r;
                this.loading = false;
            });
    }
}
