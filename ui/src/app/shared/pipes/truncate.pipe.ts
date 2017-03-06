import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'truncate'
})
export class TruncatePipe implements PipeTransform {
    transform(value: string, args: string): string {
        return value.length > Number(args) ? value.substring(0, Number(args)) + '...' : value;
    }
}
