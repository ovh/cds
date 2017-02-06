import {Observable, BehaviorSubject} from 'rxjs/Rx';
import {WorkerResponse} from './worker.model';
export class CDSWorker {

    // Webworker
    sharedWorker: any = null;
    webWorkerId: number;
    webWorker: Worker = null;

    sharedWorkerScript: string;
    webWorkerScript: string;

    private _response: BehaviorSubject<WorkerResponse> = new BehaviorSubject(new WorkerResponse());

    constructor(sharedWorkerScript: string, webWorkerScript: string) {
        this.sharedWorkerScript = sharedWorkerScript;
        this.webWorkerScript = webWorkerScript;
    }

    response(): Observable<WorkerResponse> {
        return new Observable<WorkerResponse>(fn => this._response.subscribe(fn));
    }

    /**
     * Create worker
     * @param msgToSend Message to send to worker to start it.
     */
    start(msgToSend: any) {
        // Check Shared worker compatibility
        if (typeof (SharedWorker) !== 'undefined') {
            // If no worker exist
            if (!this.sharedWorker) {
                this.sharedWorker = new SharedWorker(this.sharedWorkerScript);

                // Message received from worker
                this.sharedWorker.port.onmessage = ((e) => {
                    if (e.data && e.data !== 'null') {
                        let response: WorkerResponse = new WorkerResponse();
                        // if ID, save id and send message to worker
                        if (e.data.worker_id) {
                            response.worker_id = e.data.worker_id;
                            this.webWorkerId = e.data.worker_id;
                            this.updateWorker('subscribe', msgToSend);
                        } else {
                            response.data = e.data;
                        }
                        this._response.next(response);
                    }
                });
                // On Error : log
                this.sharedWorker.port.onerror = function (e) {
                    console.log('Warning Worker Error: ', e);
                };
            } else {
                // If worker already exist, send unsubscribe message to worker to remove it from current polling
                this.updateWorker('unsubscribe', {});
                // Call worker with new datas to poll
                this.updateWorker('subscribe', msgToSend);
            }
        } else {
            // Use web worker for safari, and edge. Web Workers are not shared between tabs
            if (!this.webWorker) {
                this.webWorker = new Worker(this.webWorkerScript);
                this.webWorker.onmessage = ((e) => {
                    if (e.data && e.data !== 'null') {
                        this._response.next(e.data);
                    }
                });
                this.webWorker.onerror = function (e) {
                    console.log('Warning Worker Error: ', e);
                };
                this.updateWorker('subscribe', msgToSend);
            } else {
                // If worker exist, delete it and start a new one
                this.updateWorker('unsubscribe');
                this.start(msgToSend);
            }
        }
    }

    updateWorker(action: string, msgToSend?: any): void {
        if (msgToSend) {
            msgToSend.action = msgToSend.action = action;
        }
        if (typeof (SharedWorker) !== 'undefined' && this.sharedWorker) {
            if (this.webWorkerId > 0) {
                msgToSend.id = this.webWorkerId;
                this.sharedWorker.port.postMessage(msgToSend);
            }
        } else {
            if (action === 'subscribe') {
                if (this.webWorker) {
                    this.updateWorker('unsubscribe');
                    this.start(msgToSend);
                }
                this.webWorker.postMessage(msgToSend);

            } else if (action === 'unsubscribe') {
                this.webWorker.terminate();
                this.webWorker = null;
            }
        }
    }
}
