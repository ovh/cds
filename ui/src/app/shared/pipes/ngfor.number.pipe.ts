import {Pipe, PipeTransform} from '@angular/core';

@Pipe({name: 'ngForNumber'})
export class NgForNumber implements PipeTransform {
    transform(value): any {
        let res = [];
        for (let i = 0; i < value; i++) {
            res.push(i + 1);
        }
        return res;
    }
}
