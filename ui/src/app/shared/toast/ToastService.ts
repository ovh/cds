import {Injectable} from '@angular/core';
import {ToasterService} from 'angular2-toaster/angular2-toaster';

@Injectable()
export class ToastService {

    constructor(private _toasterService: ToasterService) {

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
