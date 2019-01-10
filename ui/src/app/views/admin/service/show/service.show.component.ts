import { Component, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { CodemirrorComponent } from 'ng2-codemirror-typescript';
import { Service } from '../../../../model/service.model';
import { ServiceService } from '../../../../service/services.module';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';

@Component({
    selector: 'app-service-show',
    templateUrl: './service.show.html',
    styleUrls: ['./service.show.scss']
})
export class ServiceShowComponent {
    loading: boolean;
    service: Service;
    codeMirrorConfig: any;
    config: any;
    status: string;

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

    path: Array<PathItem>;

    constructor(
        private _serviceService: ServiceService,
        private _route: ActivatedRoute
    ) {
        this.codeMirrorConfig = this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: true
        };

        this._route.params.subscribe(params => {
            const name = params['name'];
            this.loading = true;
            this._serviceService.getService(name).subscribe(srv => {
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
