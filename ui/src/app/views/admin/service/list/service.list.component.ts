import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { MonitoringStatus, MonitoringStatusLine } from 'app/model/monitoring.model';
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
    filteredStatusLines: Array<MonitoringStatusLine>;
    globals: Array<Global> = [];
    globalStatus: Global;
    globalVersion: Global;
    path: Array<PathItem>;

    refreshStatus(): Observable<any> {
        return this._monitoringService.getStatus().pipe(tap(r => {
            this.status = r;
            this.filterChange();
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

    constructor(
        private _monitoringService: MonitoringService,
        private _serviceService: ServiceService,
        private _cd: ChangeDetectorRef
    ) {
        this.loading = true;

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
