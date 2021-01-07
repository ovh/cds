import {Injectable} from '@angular/core';

@Injectable()
export class SharedService {

    /**
     * Get the height for a textarea.
     *
     * @param value Value to display
     * @returns
     */
    getTextAreaheight(value: string): number {
        let size = 0;
        if (value) {
            size = value.split('\n').length ;
        }
        if (size === 0) {
            size++;
        }
        return size;
    }
}
