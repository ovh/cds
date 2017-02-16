import { Injectable } from '@angular/core';

declare var Duration: any;

@Injectable()
export class DurationService {

    constructor() { }

    duration(from: Date, to: Date): string {
        let fromMs = Math.round(from.getTime() / 1000) * 1000;
        let toMs = Math.round(to.getTime() / 1000) * 1000;
        return (new Duration(toMs - fromMs)).toString();
    }

}
