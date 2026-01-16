import {Pipe, PipeTransform} from '@angular/core';

@Pipe({
    name: 'truncate',
    standalone: false
})
export class TruncatePipe implements PipeTransform {
    transform(value: string, args: string): string {
        return value.length > Number(args) ? value.substring(0, Number(args)) + '...' : value;
    }
}
