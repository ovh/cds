import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { MonitoringStatusLine, MonitoringStatusLineUtil } from 'app/model/monitoring.model';
import { Service } from 'app/model/service.model';
import { ServiceService } from 'app/service/service/service.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-service-show',
    templateUrl: './service.show.html',
    styleUrls: ['./service.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ServiceShowComponent implements OnInit, OnDestroy {
    @ViewChild('textareaCodeMirror') codemirror: any;
    loading: boolean;
    columns: Array<Column<MonitoringStatusLine>>;
    filteredStatusLines: Filter<MonitoringStatusLine>;
    filter = 'NOTICE';
    service: Service;
    codeMirrorConfig: any;
    config: any;
    status: string;
    path: Array<PathItem>;
    themeSubscription: Subscription;

    constructor(
        private _serviceService: ServiceService,
        private _route: ActivatedRoute,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: true
        };

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
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            this._cd.markForCheck();
        });

        this._route.params.subscribe(params => {
            const name = params['name'];
            this.loading = true;
            this._cd.markForCheck();
            this._serviceService.getService(name).pipe(finalize(() => this._cd.markForCheck())).subscribe(srv => {
                this.loading = false;
                this.service = srv;
                this.config = JSON.stringify(srv.config, null, 4);
                srv.status = 'OK';
                if (srv.monitoring_status.lines) {
                    for (let index = 0; index < srv.monitoring_status.lines.length; index++) {
                        const element = srv.monitoring_status.lines[index];
                        if (element.component === srv.name + '/Version') {
                            this.service.version = element.value;
                        }
                        if (srv.status !== 'AL') {
                            if (element.status === 'AL' || element.status === 'WARN') {
                                srv.status = element.status;
                            }
                        }
                    }
                }
                this.updatePath();
            });
        });
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'services_list',
            routerLink: ['/', 'admin', 'services']
        }];

        if (this.service) {
            this.path.push(<PathItem>{
                text: this.service.type + ' - ' + this.service.name,
                routerLink: ['/', 'admin', 'services', this.service.name]
            });
        }
    }
}
