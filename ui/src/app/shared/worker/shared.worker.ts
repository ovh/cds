import {BehaviorSubject, Observable} from 'rxjs';
import {environment} from '../../../environments/environment';
import {CDSWorker} from './worker';

declare module SharedWorker {
    interface AbstractWorker extends EventTarget {
        onerror: (ev: ErrorEvent) => any;
    }
    export interface SharedWorker extends AbstractWorker {
        port: MessagePort;
        onconnect: (messageEvent: MessageEvent) => void;
    }
}
declare var SharedWorker: {
    prototype: SharedWorker.SharedWorker;
    new(stringUrl: string, name?: string): SharedWorker.SharedWorker;
};

export class CDSSharedWorker implements CDSWorker {

    // Webworker
    sharedWorker: SharedWorker.SharedWorker;

    sharedWorkerScript: string;

    private _response: BehaviorSubject<any> = new BehaviorSubject(null);

    constructor(sharedWorkerScript: string) {
        this.sharedWorkerScript = sharedWorkerScript;
    }

    response(): Observable<any> {
        return new Observable<any>(fn => this._response.subscribe(fn));
    }

    /**
     * Create worker
     * @param msgToSend Message to send to worker to start it.
     */
    start(msgToSend: any) {
        if (!this.sharedWorker) {
            this.sharedWorker = new SharedWorker(this.sharedWorkerScript, 'cds-' + environment.name);
            this.sharedWorker.port.onmessage = ((e) => {
                if (e.data && e.data !== 'null') {
                    this._response.next(e.data);
                }
            });
            this.sharedWorker.onerror = function (e) {
                console.log('Worker Error: ', e);
            };
            this.sharedWorker.port.start();
            this.sharedWorker.port.postMessage(msgToSend);
        } else {
            // If worker exist, delete it and start a new one
            this.stop();
            this.start(msgToSend);
        }
    }

    sendMsg(msgToSend: any) {
        if (this.sharedWorker) {
            this.sharedWorker.port.postMessage(msgToSend);
        }
    }

    stop() {
        if (this.sharedWorker) {
            this.sharedWorker.port.close();
            this.sharedWorker = null;
        }
    }
}
