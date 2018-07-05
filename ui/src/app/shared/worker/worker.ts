import {Observable} from 'rxjs/index';

export interface CDSWorker {

    start(msg: any);
    response(): Observable<any>;
    sendMsg(msgToSend: any);
    stop();
}
