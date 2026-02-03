import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { concatMap, delay, filter, retryWhen } from 'rxjs/operators';
import { WebSocketSubject, webSocket } from 'rxjs/webSocket';
import { WebsocketV2Event, WebsocketV2Filter, WebsocketV2FilterType } from './model/websocket-v2';
import { Store } from '@ngxs/store';
import { AddEventV2 } from './store/event-v2.action';
import { NzMessageService } from 'ng-zorro-antd/message';

@Injectable()
export class EventV2Service {

    websocket: WebSocketSubject<any>;
    currentFilters: Array<WebsocketV2Filter>;
    private connected: boolean;

    constructor(
        private _router: Router,
        private _messageService: NzMessageService,
        private _store: Store
    ) { }

    stopWebsocket() {
        if (this.websocket) {
            this.websocket.complete();
        }
    }

    startWebsocket() {
        const protocol = window.location.protocol.replace('http', 'ws');
        const host = window.location.host;
        const href = this._router['location']._basePath;

        this.websocket = webSocket({
            url: `${protocol}//${host}${href}/cdsapi/v2/ws`,
            openObserver: {
                next: value => {
                    if (value.type === 'open') {
                        this.connected = true;
                        if (this.currentFilters) {
                            this.websocket.next(this.currentFilters);
                        }
                    }
                }
            }
        });

        this.websocket
            .pipe(retryWhen(errors => errors.pipe(delay(2000))))
            .pipe(
                filter((message: WebsocketV2Event): boolean => {
                    let ok = message.status === 'OK';
                    if (!ok) {
                        this._messageService.error(message.error);
                    }
                    return ok;
                }),
                concatMap((message: WebsocketV2Event) => this._store.dispatch(new AddEventV2(message.event))),
            ).subscribe((e) => { console.log(e) }, (err) => {
                console.error('Error: ', err);
            }, () => {
                console.warn('Websocket Completed');
            });
    }

    subscribeAutoFromPath(path: string) {
        // When we move from a page to another we reset the filters
        let fs: Array<WebsocketV2Filter> = [
            <WebsocketV2Filter>{ type: WebsocketV2FilterType.GLOBAL }
        ];
        let pathSplitted = path.substring(1, path.length).split('/');
        switch (pathSplitted[0]) {
            case 'settings':
                if (pathSplitted.length === 1) { // Ignore settings root page
                    break;
                }
                let pageName = pathSplitted[1];
                switch (pageName) {
                    case 'queue':
                        fs.push(<WebsocketV2Filter>{
                            type: WebsocketV2FilterType.QUEUE
                        });
                        break;
                }
                break;
            case 'project':
                if (pathSplitted.length === 1) { // Ignore project creation page
                    break;
                }
                let projectKey = pathSplitted[1];
                fs.push(<WebsocketV2Filter>{
                    type: WebsocketV2FilterType.PROJECT,
                    project_key: projectKey
                });
                break;
        }
        this.updateFilters(fs);
    }

    updateFilters(fs: Array<WebsocketV2Filter>): void {
        this.currentFilters = fs;
        if (this.connected) {
            this.websocket.next(this.currentFilters);
        }
    }

    updateFilter(filter: WebsocketV2Filter): void {
        this.currentFilters = this.currentFilters.filter(f => f.type !== filter.type).concat(filter);
        if (this.connected) {
            this.websocket.next(this.currentFilters);
        }
    }

}
