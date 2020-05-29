import { Injectable } from '@angular/core';
import { ToasterConfig, ToasterService } from 'angular2-toaster/angular2-toaster';

@Injectable()
export class ToastService {
    private configDefault: ToasterConfig = new ToasterConfig({
        mouseoverTimerStop: true,
        toastContainerId: 1
    });
    private configErrorHTTP: ToasterConfig = new ToasterConfig({
        showCloseButton: true,
        timeout: 0,
        toastContainerId: 2
    });

    constructor(private _toasterService: ToasterService) {
    }

    getConfigDefault(): ToasterConfig {
        return this.configDefault;
    }

    getConfigErrorHTTP(): ToasterConfig {
        return this.configErrorHTTP;
    }

    success(title: string, msg: string) {
        this._toasterService.pop('success', title, msg);
    }

    info(title: string, msg: string) {
        this._toasterService.pop('info', title, msg);
    }

    error(title: string, msg: string) {
        this._toasterService.pop(
            { type: 'error', title: title, body: msg, toastContainerId: 1 }
        );
    }

    errorHTTP(title: string, msg: string, from: string, requestID: string) {
        let body = from ? `${msg} (from: ${from})` : msg;
        this._toasterService.pop(
            { type: 'error', title: title, body: body, toastContainerId: 2 }
        );
    }
}
