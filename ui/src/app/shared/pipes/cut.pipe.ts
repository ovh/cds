import {Pipe, PipeTransform} from '@angular/core';

@Pipe({name: 'cut'})
export class CutPipe implements PipeTransform {
    transform(value: string, args: string): any {
        return value.substr(0, Number(args));
    }
}
