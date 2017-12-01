import {Observable} from 'rxjs/Observable';
import {BehaviorSubject} from 'rxjs/BehaviorSubject'
export class CDSWorker {

    // Webworker
    webWorker: Worker = null;

    webWorkerScript: string;

    private _response: BehaviorSubject<string> = new BehaviorSubject(null);

    constructor(webWorkerScript: string) {
        this.webWorkerScript = webWorkerScript;
    }

    response(): Observable<string> {
        return new Observable<string>(fn => this._response.subscribe(fn));
    }

    /**
     * Create worker
     * @param msgToSend Message to send to worker to start it.
     */
    start(msgToSend: any) {
        // Use web worker for safari, and edge. Web Workers are not shared between tabs
        if (!this.webWorker) {
            this.webWorker = new Worker(this.webWorkerScript);
            this.webWorker.onmessage = ((e) => {
                if (e.data && e.data !== 'null') {
                    this._response.next(e.data);
                }
            });
            this.webWorker.onerror = function (e) {
                console.log('Worker Error: ', e);
            };
            this.webWorker.postMessage(msgToSend);
        } else {
            // If worker exist, delete it and start a new one
            this.stop();
            this.start(msgToSend);
        }
    }

    stop() {
        if (this.webWorker) {
            this.webWorker.terminate();
            this.webWorker = null;
        }
    }
}
