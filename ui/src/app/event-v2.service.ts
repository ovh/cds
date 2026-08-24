import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { concatMap, delay, filter, retryWhen } from 'rxjs/operators';
import { WebSocketSubject, webSocket } from 'rxjs/webSocket';
import { WebsocketV2Event, WebsocketV2Filter, WebsocketV2FilterType } from './model/websocket-v2';
import { Store } from '@ngxs/store';
import { AddEventV2 } from './store/event-v2.action';
import { NzMessageService } from 'ng-zorro-antd/message';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable()
export class EventV2Service {

    websocket: WebSocketSubject<any>;
    currentFilters: Array<WebsocketV2Filter>;
    private connected: boolean;
    private _connected = new BehaviorSubject<boolean>(false);

    constructor(
        private _router: Router,
        private _messageService: NzMessageService,
        private _store: Store
    ) { }

    /**
     * Emits whenever the websocket opens or closes. Events sent while it was closed are lost, so a
     * view that mirrors a state through events has to refresh itself when it opens again.
     */
    get connected$(): Observable<boolean> {
        return this._connected.asObservable();
    }

    stopWebsocket() {
        if (this.websocket) {
            this.websocket.complete();
        }
        this.setConnected(false);
    }

    private setConnected(connected: boolean): void {
        this.connected = connected;
        if (this._connected.value !== connected) {
            this._connected.next(connected);
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
                        if (this.currentFilters) {
                            this.websocket.next(this.currentFilters);
                        }
                        this.setConnected(true);
                    }
                }
            },
            closeObserver: {
                next: () => {
                    this.setConnected(false);
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
            ).subscribe(() => { }, (err) => {
                this.setConnected(false);
                console.error('Error: ', err);
            }, () => {
                this.setConnected(false);
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
