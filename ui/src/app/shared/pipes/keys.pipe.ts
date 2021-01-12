import { Pipe, PipeTransform } from '@angular/core';

@Pipe({name: 'keys'})
export class KeysPipe implements PipeTransform {
    transform(value): any {
        return Object.keys(value);
    }
}
