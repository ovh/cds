import { LiveAnnouncer } from '@angular/cdk/a11y';
import { Injectable, inject } from '@angular/core';
import { Store } from '@ngxs/store';
import { HelpState } from 'app/store/help.state';
import { filter } from 'rxjs/operators';
import { NzNotificationService } from 'ng-zorro-antd/notification';
import { BehaviorSubject, Observable } from 'rxjs';
import { NzNotificationDataOptions } from 'ng-zorro-antd/notification';

export class ToastHTTPErrorData {
    status: number;
    from: string;
    request_id: string;
    help: string;
    title: string
}


@Injectable()
export class ToastService {
    private _toastQueue: BehaviorSubject<NzNotificationDataOptions<ToastHTTPErrorData>> = new BehaviorSubject(null);

    _nzNotificationService = inject(NzNotificationService);
    _store = inject(Store)
    _liveAnnouncer = inject(LiveAnnouncer);

    constructor() { }

    getObservable(): Observable<NzNotificationDataOptions<ToastHTTPErrorData>> {
        return new Observable<NzNotificationDataOptions<ToastHTTPErrorData>>(fn => this._toastQueue.subscribe(fn));
    }

    success(title: string, msg: string) {
        this._nzNotificationService.success(title, msg);
        this._liveAnnouncer.announce(msg ? `${title}. ${msg}` : title, 'polite');
    }

    info(title: string, msg: string) {
        this._nzNotificationService.info(title, msg);
        this._liveAnnouncer.announce(msg ? `${title}. ${msg}` : title, 'polite');
    }

    error(title: string, msg: string) {
        this._nzNotificationService.error(title, msg);
        this._liveAnnouncer.announce(msg ? `Error: ${title}. ${msg}` : `Error: ${title}`, 'assertive');
    }

    errorHTTP(status: number, message: string, from: string, request_id: string) {
        this._liveAnnouncer.announce(`Error: ${message}`, 'assertive');
        this._store.select(HelpState.last).pipe(
            filter((help) => help != null),
        ).subscribe(help => {
            this._toastQueue.next({
                nzPauseOnHover: true,
                nzDuration: status === 500 ? 0 : 3000,
                nzData: {
                    status,
                    from,
                    request_id,
                    help: help.error,
                    title: message,
                }
            });
        });
    }
}

