import { Injectable } from '@angular/core';
import { BodyOutputType, ToasterConfig, ToasterService } from 'angular2-toaster/angular2-toaster';
import { HelpService } from 'app/service/services.module';
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
        private _helpService: HelpService
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
            { type: 'success', title: title, body: msg, toastContainerId: 1 }
        );
    }

    info(title: string, msg: string) {
        this._toasterService.pop(
            { type: 'info', title: title, body: msg, toastContainerId: 1 }
        );
    }

    error(title: string, msg: string) {
        this._toasterService.pop(
            { type: 'error', title: title, body: msg, toastContainerId: 1 }
        );
    }

    errorHTTP(status: number, message: string, from: string, request_id: string) {
        this._helpService.getHelp().subscribe(c => {
            this._toasterService.pop(
                {
                    type: 'error',
                    title: message,
                    body: ToastHTTPErrorComponent,
                    bodyOutputType: BodyOutputType.Component,
                    toastContainerId: status < 500 ? 2 : 3,
                    data: <ToastHTTPErrorData>{
                        status: status,
                        from: from,
                        request_id: request_id,
                        help: c.error
                    }
                }
            );
        });
    }
}

