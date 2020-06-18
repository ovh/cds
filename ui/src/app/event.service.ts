import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { AppService } from 'app/app.service';
import { AuthentifiedUser } from 'app/model/user.model';
import { WebSocketEvent, WebSocketMessage } from 'app/model/websocket.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { WebSocketSubject } from 'rxjs/internal-compatibility';
import { delay, retryWhen } from 'rxjs/operators';
import { webSocket } from 'rxjs/webSocket';

@Injectable()
export class EventService {

    websocket: WebSocketSubject<any>;
    currentFilter: WebSocketMessage;
    private connected: boolean;

    constructor(
        private _router: Router,
        private _appService: AppService,
        private _toastService: ToastService
    ) {}

    stopWebsocket() {
        if (this.websocket) {
            this.websocket.complete();
        }
    }
    startWebsocket() {
        const protocol = window.location.protocol.replace('http', 'ws');
        const host = window.location.host;
        const href = this._router['location']._baseHref;

        this.websocket = webSocket({
            url: `${protocol}//${host}${href}/cdsapi/ws`,
            openObserver: {
                next: value => {
                    if (value.type === 'open') {
                        this.connected = true;
                        if (this.currentFilter) {
                            this.websocket.next(this.currentFilter);
                        }
                    }
                }
            }
        });

        this.websocket
            .pipe(retryWhen(errors => errors.pipe(delay(2000))))
            .subscribe((message: WebSocketEvent) => {
                if (message.status === 'OK') {
                    this._appService.manageEvent(message.event);
                } else {
                    this._toastService.error('', message.error);
                }
            }, (err) => {
                console.error('Error: ', err)
            }, () => {
                console.warn('Websocket Completed');
            });
    }

    addOperationFilter(uuid: string) {
        this.currentFilter.operation = uuid;
        this.websocket.next(this.currentFilter);
    }

    updateFilter(f: WebSocketMessage): void {
        this.currentFilter = f;
        if (this.connected) {
            this.websocket.next(this.currentFilter);
        }
    }

    manageWebsocketFilterByUrl(url: string) {
        let msg =  new WebSocketMessage();
        let urlSplitted = url.substr(1, url.length - 1).split('/');
        switch (urlSplitted[0]) {
            case 'home':
                msg.favorites = true;
                break;
            case 'project':
                switch (urlSplitted.length) {
                    case 1: // project creation
                        break;
                    case 2: // project view
                        msg.project_key = urlSplitted[1].split('?')[0];
                        msg.type = 'project';
                        break;
                    default: // App/pipeline/env/workflow view
                        msg.project_key = urlSplitted[1].split('?')[0];
                        this.manageWebsocketFilterProjectPath(urlSplitted, msg);
                }
                break;
            case 'settings':
                if (urlSplitted.length === 2 && urlSplitted[1] === 'queue') {
                    msg.queue = true;
                }
                break;
        }
        this.updateFilter(msg);
    }

    manageWebsocketFilterProjectPath(urlSplitted: Array<string>, msg: WebSocketMessage) {
        switch (urlSplitted[2]) {
            case 'pipeline':
                if (urlSplitted.length >= 4) {
                    msg.pipeline_name = urlSplitted[3].split('?')[0];
                    msg.type = 'pipeline';
                }
                break;
            case 'application':
                if (urlSplitted.length >= 4) {
                    msg.application_name = urlSplitted[3].split('?')[0];
                    msg.type = 'application';
                }
                break;
            case 'environment':
                if (urlSplitted.length >= 4) {
                    msg.environment_name = urlSplitted[3].split('?')[0];
                    msg.type = 'environment';
                }
                break;
            case 'workflow':
                if (urlSplitted.length >= 4) {
                    msg.workflow_name = urlSplitted[3].split('?')[0];
                    msg.type = 'workflow';
                }
                if (urlSplitted.length >= 6) {
                    msg.workflow_run_num = Number(urlSplitted[5].split('?')[0]);
                    msg.type = 'workflow';
                }
                if (urlSplitted.length >= 8) {
                    msg.workflow_node_run_id = Number(urlSplitted[7].split('?')[0]);
                    msg.type = 'workflow';
                }
                break;
        }
    }
}
