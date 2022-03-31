import { Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { HelpState } from 'app/store/help.state';
import { filter } from 'rxjs/operators';
import { NzNotificationService } from 'ng-zorro-antd/notification';
import { BehaviorSubject, Observable } from 'rxjs';
import { NzNotificationDataOptions } from 'ng-zorro-antd/notification/typings';

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

    constructor(
        private _nzNotificationService: NzNotificationService,
        private _store: Store
    ) {}

    getObservable(): Observable<NzNotificationDataOptions<ToastHTTPErrorData>> {
        return new Observable<NzNotificationDataOptions<ToastHTTPErrorData>>(fn => this._toastQueue.subscribe(fn));
    }

    success(title: string, msg: string) {
        this._nzNotificationService.success(title, msg);
    }

    info(title: string, msg: string) {
        this._nzNotificationService.info(title, msg);
    }

    error(title: string, msg: string) {
        this._nzNotificationService.error(title, msg);
    }

    errorHTTP(status: number, message: string, from: string, request_id: string) {
        this._store.select(HelpState.last)
        .pipe(
            filter((help) => help != null),
        )
        .subscribe(help => {
            this._toastQueue.next( {
                nzPauseOnHover: true,
                nzDuration: status < 500 ? 3000 : 0,
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

