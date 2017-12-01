import {Pipe, PipeTransform} from '@angular/core';

@Pipe({name: 'forMap'})
export class ForMapPipe implements PipeTransform {
    transform(m: Map<any, any>): Array<{key, value}> {
        let listkeyValue = new Array<{key, value}>();
        m.forEach((v, k) => {
            listkeyValue.push({key: k, value: v});
        });
        return listkeyValue;
    }
}
