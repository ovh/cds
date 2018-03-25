import {Injectable} from '@angular/core';

@Injectable()
export class InfoLevelService {
    private infos = [
        { 'name': 'info', 'value': 'info' },
        { 'name': 'warning', 'value': 'warning' }
    ];

    /**
     * Get infos levels list
     */
    getInfoLevels() {
        return this.infos;
    }
}
