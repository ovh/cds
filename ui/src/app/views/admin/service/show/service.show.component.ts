import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Service } from 'app/model/service.model';
import { ServiceService, ThemeStore } from 'app/service/services.module';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-service-show',
    templateUrl: './service.show.html',
    styleUrls: ['./service.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ServiceShowComponent implements OnInit {
    @ViewChild('textareaCodeMirror', {static: false}) codemirror: any;

    loading: boolean;
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
    }

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
                            if (element.status === 'AL') {
                                srv.status = element.status;
                            } else if (element.status === 'WARN') {
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
