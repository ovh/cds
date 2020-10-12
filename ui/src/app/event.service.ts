import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { AppService } from 'app/app.service';
import { WebsocketEvent, WebsocketFilter, WebsocketFilterType } from 'app/model/websocket.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { WebSocketSubject } from 'rxjs/internal-compatibility';
import { delay, retryWhen } from 'rxjs/operators';
import { webSocket } from 'rxjs/webSocket';

@Injectable()
export class EventService {

    websocket: WebSocketSubject<any>;
    currentFilters: Array<WebsocketFilter>;
    private connected: boolean;

    constructor(
        private _router: Router,
        private _appService: AppService,
        private _toastService: ToastService
    ) { }

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
                        if (this.currentFilters) {
                            this.websocket.next(this.currentFilters);
                        }
                    }
                }
            }
        });

        this.websocket
            .pipe(retryWhen(errors => errors.pipe(delay(2000))))
            .subscribe((message: WebsocketEvent) => {
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

    updateFilters(fs: Array<WebsocketFilter>): void {
        this.currentFilters = fs;
        if (this.connected) {
            this.websocket.next(this.currentFilters);
        }
    }

    subscribeToWorkflowPurgeDryRun(projectKey: string, workflowName: string) {
        this.updateFilters(this.currentFilters.concat(<WebsocketFilter>{
            type: WebsocketFilterType.WORKFLOW_RETENTION_DRYRUN,
            project_key: projectKey,
            workflow_name: workflowName
        }));
    }

    unsubscribeWorkflowRetention() {
        this.updateFilters(this.currentFilters.filter(f => f.type !== WebsocketFilterType.WORKFLOW_RETENTION_DRYRUN));
    }

    subscribeToOperation(projectKey: string, operationUUID: string) {
        this.updateFilters(this.currentFilters.concat(<WebsocketFilter>{
            type: WebsocketFilterType.OPERATION,
            project_key: projectKey,
            operation_uuid: operationUUID
        }));
    }

    subscribeAutoFromUrl(url: string) {
        // When we move from a page to another we reset the filters
        let fs: Array<WebsocketFilter> = [
            <WebsocketFilter>{ type: WebsocketFilterType.GLOBAL }
        ];

        let urlSplitted = url.substr(1, url.length - 1).split('/');
        switch (urlSplitted[0]) {
            case 'home':
                fs.push(<WebsocketFilter>{ type: WebsocketFilterType.TIMELINE });
                break;
            case 'settings':
                if (urlSplitted.length === 1) { // Ignore settings root page
                    break;
                }
                let pageName = urlSplitted[1];
                switch (pageName) {
                    case 'queue':
                        fs.push(<WebsocketFilter>{ type: WebsocketFilterType.QUEUE });
                        break;
                }
                break;
            case 'project':
                if (urlSplitted.length === 1) { // Ignore project creation page
                    break;
                }
                let projectKey = urlSplitted[1].split('?')[0];
                if (urlSplitted.length === 2) { // Project page
                    fs.push(<WebsocketFilter>{
                        type: WebsocketFilterType.PROJECT,
                        project_key: projectKey
                    });
                    break;
                }
                if (urlSplitted.length === 3) { // Ignore application/pipeline/environment/workflow creation pages
                    break
                }
                let entityType = urlSplitted[2];
                let entityName = urlSplitted[3].split('?')[0];
                switch (entityType) {
                    case 'pipeline':
                        fs.push(<WebsocketFilter>{
                            type: WebsocketFilterType.PIPELINE,
                            project_key: projectKey,
                            pipeline_name: entityName
                        });
                        break;
                    case 'application':
                        fs.push(<WebsocketFilter>{
                            type: WebsocketFilterType.APPLICATION,
                            project_key: projectKey,
                            application_name: entityName
                        });
                        break;
                    case 'environment':
                        fs.push(<WebsocketFilter>{
                            type: WebsocketFilterType.ENVIRONMENT,
                            project_key: projectKey,
                            environment_name: entityName
                        });
                        break;
                    case 'workflow':
                        fs.push(<WebsocketFilter>{
                            type: WebsocketFilterType.WORKFLOW,
                            project_key: projectKey,
                            workflow_name: entityName
                        }, <WebsocketFilter>{
                            type: WebsocketFilterType.ASCODE_EVENT,
                            project_key: projectKey,
                            workflow_name: entityName
                        });
                        if (urlSplitted.length >= 6) {
                            fs.push(<WebsocketFilter>{
                                type: WebsocketFilterType.WORKFLOW_RUN,
                                project_key: projectKey,
                                workflow_name: entityName,
                                workflow_run_num: Number(urlSplitted[5].split('?')[0])
                            });
                        }
                        if (urlSplitted.length >= 8) {
                            fs.push(<WebsocketFilter>{
                                type: WebsocketFilterType.WORKFLOW_NODE_RUN,
                                project_key: projectKey,
                                workflow_name: entityName,
                                workflow_node_run_id: Number(urlSplitted[7].split('?')[0])
                            });
                        }
                        break;
                }
                break;
        }

        this.updateFilters(fs);
    }

}
