import { Component } from '@angular/core';
import { MonitoringStatus, MonitoringStatusLine } from 'app/model/monitoring.model';
import { Global } from '../../../model/service.model';
import { MonitoringService } from '../../../service/monitoring/monitoring.service';
import { ServiceService } from '../../../service/service/service.service';


@Component({
    selector: 'app-services',
    templateUrl: './services.html',
    styleUrls: ['./services.scss']
})
export class ServicesComponent {
    loading = false;

    filter = 'NOTICE';
    status: MonitoringStatus;

    filteredStatusLines: Array<MonitoringStatusLine>;

    globals: Array<Global> = [];

    constructor(private _monitoringService: MonitoringService,
                private _serviceService: ServiceService) {
        this.loading = true;
        this._monitoringService.getStatus()
            .subscribe(r => {
                this.status = r;
                this.filterChange();
                this._serviceService.getServices()
                    .subscribe((services) => {
                        if (services) {
                            services.forEach(s => {
                                s.status = 'OK';
                                if (s.monitoring_status.lines) {
                                    for (let index = 0; index < s.monitoring_status.lines.length; index++) {
                                        const element = s.monitoring_status.lines[index];
                                        if (element.status === 'AL') {
                                            s.status = element.status;
                                            break
                                        } else if (element.status === 'WARN') {
                                            s.status = element.status;
                                        }
                                    }
                                }
                            })
                            r.lines.forEach(g => {
                                if (g.component.startsWith('Global/')) {
                                    let type = g.component.slice(7);
                                    let global = new Global();
                                    global.name = type;
                                    global.value = g.value;
                                    global.status = g.status;
                                    global.services = [];
                                    global.services = services.filter((srv) => { return srv.type === type})
                                    this.globals.push(global);
                                }
                            });
                            this.loading = false;
                        }
                    });
            });
    }

    filterChange(): void {
        if (!this.filter) {
            this.filteredStatusLines = this.status.lines;
            return;
        }

        if (this.filter === 'NOTICE') {
            this.filteredStatusLines = this.status.lines.filter(line => {
                return line.status.indexOf('AL') !== -1 || line.status.indexOf('WARN') !== -1
            });
            return
        }

        const lowerFilter = this.filter.toLowerCase();

        this.filteredStatusLines = this.status.lines.filter(line => {
            return line.status.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.component.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.value.toLowerCase().indexOf(lowerFilter) !== -1
        });
    }

}
