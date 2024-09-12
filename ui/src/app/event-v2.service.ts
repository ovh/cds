import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { ToastService } from 'app/shared/toast/ToastService';
import { concatMap, delay, filter, retryWhen } from 'rxjs/operators';
import { WebSocketSubject, webSocket } from 'rxjs/webSocket';
import { WebsocketV2Event, WebsocketV2Filter, WebsocketV2FilterType } from './model/websocket-v2';
import { Store } from '@ngxs/store';
import { AddEventV2 } from './store/event-v2.action';
import { FeatureNames, FeatureService } from './service/feature/feature.service';

@Injectable()
export class EventV2Service {

    websocket: WebSocketSubject<any>;
    currentFilters: Array<WebsocketV2Filter>;
    private connected: boolean;

    constructor(
        private _router: Router,
        private _toastService: ToastService,
        private _store: Store,
        private _featureService: FeatureService
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
                        this._toastService.error('', message.error);
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

    subscribeAutoFromUrl(url: string) {
        // When we move from a page to another we reset the filters
        let fs: Array<WebsocketV2Filter> = [
            <WebsocketV2Filter>{ type: WebsocketV2FilterType.GLOBAL }
        ];
        let urlSplitted = url.substr(1, url.length - 1).split('/');
        switch (urlSplitted[0]) {
            case 'project':
                if (urlSplitted.length === 1) { // Ignore project creation page
                    break;
                }

                let projectKey = urlSplitted[1].split('?')[0];

                this._featureService.isEnabled(FeatureNames.AllAsCode, { project_key: projectKey }).subscribe(f => {
                    if (f.enabled) {
                        fs.push(<WebsocketV2Filter>{
                            type: WebsocketV2FilterType.PROJECT,
                            project_key: projectKey
                        });
                    }
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
