import { Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { BodyOutputType, ToasterConfig, ToasterService } from 'angular2-toaster-sgu';
import { HelpState } from 'app/store/help.state';
import { filter } from 'rxjs/operators';
import { ToastHTTPErrorComponent, ToastHTTPErrorData } from './toast-http-error.component';

@Injectable()
export class ToastService {
    private configDefault: ToasterConfig = new ToasterConfig({
        mouseoverTimerStop: true,
        toastContainerId: 1
    });
    private configErrorHTTP: ToasterConfig = new ToasterConfig({
        mouseoverTimerStop: true,
        toastContainerId: 2
    });
    private configErrorHTTPLocked: ToasterConfig = new ToasterConfig({
        showCloseButton: true,
        tapToDismiss: false,
        timeout: 0,
        toastContainerId: 3
    });

    constructor(
        private _toasterService: ToasterService,
        private _store: Store,
    ) { }

    getConfigDefault(): ToasterConfig {
        return this.configDefault;
    }

    getConfigErrorHTTP(): ToasterConfig {
        return this.configErrorHTTP;
    }

    getConfigErrorHTTPLocked(): ToasterConfig {
        return this.configErrorHTTPLocked;
    }

    success(title: string, msg: string) {
        this._toasterService.pop(
            { type: 'success', title, body: msg, toastContainerId: 1 }
        );
    }

    info(title: string, msg: string) {
        this._toasterService.pop(
            { type: 'info', title, body: msg, toastContainerId: 1 }
        );
    }

    error(title: string, msg: string) {
        this._toasterService.pop(
            { type: 'error', title, body: msg, toastContainerId: 1 }
        );
    }

    errorHTTP(status: number, message: string, from: string, request_id: string) {
        this._store.select(HelpState.last)
        .pipe(
            filter((help) => help != null),
        )
        .subscribe(help => {
            this._toasterService.pop(
                {
                    type: 'error',
                    title: message,
                    body: ToastHTTPErrorComponent,
                    bodyOutputType: BodyOutputType.Component,
                    toastContainerId: status < 500 ? 2 : 3,
                    data: <ToastHTTPErrorData>{
                        status,
                        from,
                        request_id,
                        help: help.error
                    }
                }
            );
        });
    }
}

