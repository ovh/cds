import {Component} from '@angular/core';
import { MonitoringStatus } from 'app/model/monitoring.model';
import {MonitoringService} from '../../../service/monitoring/monitoring.service';

@Component({
    selector: 'app-status',
    templateUrl: './status.html',
    styleUrls: ['./status.scss']
})
export class StatusComponent {
    filter: string;
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

    getStatusLines() {
        if (!this.filter) {
            return this.status.lines;
        }

        const lowerFilter = this.filter.toLowerCase();

        return this.status.lines.filter(line => {
            return line.status.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.component.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.value.toLowerCase().indexOf(lowerFilter) !== -1
        });
    }
}
