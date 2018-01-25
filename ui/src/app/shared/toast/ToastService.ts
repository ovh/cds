import {Injectable} from '@angular/core';
import {ToasterService, ToasterConfig} from 'angular2-toaster/angular2-toaster';

@Injectable()
export class ToastService {
    private toasterconfig: ToasterConfig = new ToasterConfig({mouseoverTimerStop: true});

    constructor(private _toasterService: ToasterService) {
    }

    getConfig(): ToasterConfig {
      return this.toasterconfig;
    }

    success(title: string, msg: string) {
        this._toasterService.pop('success', title, msg);
    }

    info(title: string, msg: string) {
        this._toasterService.pop('info', title, msg);
    }

    error(title: string, msg: string) {
        this._toasterService.pop('error', title, msg);
    }
}
