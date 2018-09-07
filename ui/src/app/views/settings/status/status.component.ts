import {Component} from '@angular/core';
import {MonitoringStatus, MonitoringStatusLine} from 'app/model/monitoring.model';
import {MonitoringService} from '../../../service/monitoring/monitoring.service';

@Component({
    selector: 'app-status',
    templateUrl: './status.html',
    styleUrls: ['./status.scss']
})
export class StatusComponent {
    filter: string;
    status: MonitoringStatus;
    filteredStatusLines: Array<MonitoringStatusLine>;
    loading = false;

    constructor(private _monitoringService: MonitoringService) {
        this.loading = true;
        this._monitoringService.getStatus()
            .subscribe(r => {
                this.status = r;
                this.loading = false;
                this.filterChange();
            });
    }

    filterChange(): void {
        if (!this.filter) {
            this.filteredStatusLines = this.status.lines;
            return;
        }

        const lowerFilter = this.filter.toLowerCase();

        this.filteredStatusLines = this.status.lines.filter(line => {
            return line.status.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.component.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.value.toLowerCase().indexOf(lowerFilter) !== -1
        });
    }
}
