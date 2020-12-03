import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { MonitoringStatus, MonitoringStatusLine, MonitoringStatusLineUtil } from 'app/model/monitoring.model';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { forkJoin, Observable } from 'rxjs';
import { finalize, tap } from 'rxjs/operators';
import { Global, Service } from '../../../../model/service.model';
import { MonitoringService } from '../../../../service/monitoring/monitoring.service';
import { ServiceService } from '../../../../service/service/service.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';

@Component({
    selector: 'app-service-list',
    templateUrl: './service.list.html',
    styleUrls: ['./service.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ServiceListComponent {
    loading = false;
    filter = 'NOTICE';
    status: MonitoringStatus;
    services: Array<Service>;
    profiles: any;
    goroutines: any;
    columns: Array<Column<MonitoringStatusLine>>;
    filteredStatusLines: Filter<MonitoringStatusLine>;
    globals: Array<Global> = [];
    globalStatus: Global;
    globalVersion: Global;
    path: Array<PathItem>;

    constructor(
        private _monitoringService: MonitoringService,
        private _serviceService: ServiceService,
        private _cd: ChangeDetectorRef
    ) {
        this.loading = true;

        this.columns = [
            <Column<MonitoringStatusLine>>{
                name: 'common_type',
                selector: (c: MonitoringStatusLine) => c.type
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_service',
                selector: (c: MonitoringStatusLine) => c.service
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_component',
                selector: (c: MonitoringStatusLine) => c.component
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_hostname',
                selector: (c: MonitoringStatusLine) => c.hostname
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_status',
                type: ColumnType.LABEL,
                selector: (c: MonitoringStatusLine) => ({
                        class: MonitoringStatusLineUtil.color(c),
                        value: c.status
                    })
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_value',
                selector: (c: MonitoringStatusLine) => c.value
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_consumer',
                selector: (c: MonitoringStatusLine) => c.consumer
            },
            <Column<MonitoringStatusLine>>{
                name: 'common_session',
                selector: (c: MonitoringStatusLine) => c.session
            }
        ];

        this.filteredStatusLines = f => {
            const lowerFilter = f.toLowerCase();
            return (line: MonitoringStatusLine) => {
                if (f === 'NOTICE') {
                    return line.status.indexOf('AL') !== -1 || line.status.indexOf('WARN') !== -1;
                }
                if (f === 'AL' || f === 'WARN' || f === 'OK') {
                    return line.status.toLowerCase().indexOf(lowerFilter) !== -1;
                }
                return line.status.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.component.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.value.toLowerCase().indexOf(lowerFilter) !== -1 ||
                line.type.toLowerCase().indexOf(lowerFilter) !== -1 ||
                (line.service && line.service.toLowerCase().indexOf(lowerFilter) !== -1) ||
                (line.hostname && line.hostname.toLowerCase().indexOf(lowerFilter) !== -1) ||
                (line.session && line.session.toLowerCase().indexOf(lowerFilter) !== -1) ||
                (line.consumer && line.consumer.toLowerCase().indexOf(lowerFilter) !== -1);
            }
        };

        forkJoin(
            this.refreshProfiles(),
            this.refreshStatus(),
            this.refreshServices(),
            this.refreshGoroutines(),
        ).pipe(finalize(() => this._cd.markForCheck())).subscribe( _ => {
            this.status.lines.forEach(g => {
                if (g.component.startsWith('Global/')) {
                    let type = g.component.slice(7);
                    switch (type) {
                        case 'Status':
                            this.globalStatus = <Global>{
                                value: g.value,
                                name: type,
                                status: g.status
                            }
                            break;
                        case 'Version':
                            this.globalVersion = <Global>{
                                value: g.value,
                                name: type,
                                status: g.status
                            }
                            break;
                        default:
                            this.globals.push(<Global>{
                                name: type,
                                value: g.value,
                                status: g.status,
                                services: this.services.filter(srv => srv.type === type)
                            });
                            break;
                    }
                }
            });
            this.loading = false;
            this.path = [<PathItem>{
                translate: 'common_admin'
            }, <PathItem>{
                translate: 'services_list',
                routerLink: ['/', 'admin', 'services']
            }];
        })
    }

    refreshStatus(): Observable<any> {
        return this._monitoringService.getStatus().pipe(tap(r => {
            this.status = r;
        }));
    }

    refreshServices(): Observable<any> {
        return this._serviceService.getServices().pipe(tap(services => {
            if (services) {
                this.services = services;
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
            }
        }));
    }

    refreshProfiles(): Observable<any> {
        return this._monitoringService.getDebugProfiles().pipe(tap(data => {
            this.profiles = data;
        }));
    }

    refreshGoroutines(): Observable<any> {
        return this._monitoringService.getGoroutines().pipe(tap(data => {
            this.goroutines = data;
        }));
    }
}
