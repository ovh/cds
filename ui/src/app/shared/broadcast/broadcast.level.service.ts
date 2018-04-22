import {Injectable} from '@angular/core';

@Injectable()
export class BroadcastLevelService {
    private broadcasts = [
        { 'name': 'info', 'value': 'info' },
        { 'name': 'warning', 'value': 'warning' }
    ];

    /**
     * Get broadcasts levels list
     */
    getBroadcastLevels() {
        return this.broadcasts;
    }
}
