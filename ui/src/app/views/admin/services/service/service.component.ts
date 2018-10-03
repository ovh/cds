import { Component, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { CodemirrorComponent } from 'ng2-codemirror-typescript';
import { Service } from '../../../../model/service.model';
import { ServiceService } from '../../../../service/services.module';

@Component({
    selector: 'app-service',
    templateUrl: './service.html',
    styleUrls: ['./service.scss']
})
export class ServiceComponent {
    loading: boolean;
    service: Service;
    codeMirrorConfig: any;
    config: any;
    status: string;

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

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
            });
        });
    }
}
