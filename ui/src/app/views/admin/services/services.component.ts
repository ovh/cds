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
    globalQueue: Array<Global> = [];
    globalStatus: Global;
    globalVersion: Global;

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
                                    if (type === 'Status') {
                                        this.globalStatus = new Global();
                                        this.globalStatus.value = g.value;
                                        this.globalStatus.name = type;
                                        this.globalStatus.status = g.status;
                                    } else if (type === 'Version') {
                                        this.globalVersion = new Global();
                                        this.globalVersion.value = g.value;
                                        this.globalVersion.name = type;
                                        this.globalVersion.status = g.status;
                                    } else {
                                        let global = new Global();
                                        global.name = type;
                                        global.value = g.value;
                                        global.status = g.status;
                                        global.services = [];
                                        global.services = services.filter((srv) => { return srv.type === type})
                                        this.globals.push(global);
                                    }
                                }
                            });
                            this.loading = false;
                        }

                        this._monitoringService.getMetrics()
                        .subscribe(metrics => {
                            metrics.forEach(l => {
                                if (l.name !== 'queue') {
                                    return
                                }
                                l.metric.forEach(m => {
                                    m.label.forEach(lb => {
                                        if (lb.name === 'range') {
                                            let global = new Global();
                                            if (lb.value === 'all') {
                                                return;
                                            }
                                            global.name = lb.value;
                                            switch (lb.value) {
                                            case '10_less_10s':
                                                global.name = '< 10s';
                                                break;
                                            case '20_more_10s_less_30s':
                                                global.name = '< 30s';
                                                break;
                                            case '30_more_30s_less_1min':
                                                global.name = '< 1min';
                                                break;
                                            case '40_more_1min_less_2min':
                                                global.name = '< 2min';
                                                break;
                                            case '50_more_2min_less_5min':
                                                global.name = '< 5 min';
                                                break;
                                            case '60_more_5min_less_10min':
                                                global.name = '<10min';
                                                break;
                                            case '70_more_10min':
                                                global.name = '> 10min';
                                                break;
                                            default:
                                                global.name = lb.value;
                                                break;
                                            }
                                            global.value = String(m.gauge.value);
                                            global.status = 'OK';
                                            if (lb.value !== '10_less_10s' && m.gauge.value > 0) {
                                                global.status = 'WARN';
                                            }
                                            this.globalQueue.push(global);
                                        }
                                    })
                                });
                            })
                        });
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

        if (this.filter === 'AL' || this.filter === 'WARN' || this.filter === 'OK') {
            this.filteredStatusLines = this.status.lines.filter(line => {
                return line.status.indexOf(this.filter) !== -1
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
