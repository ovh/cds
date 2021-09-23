import { Pipe, PipeTransform } from '@angular/core';
import { DurationService } from '../duration/duration.service';

@Pipe({
    name: 'durationMs'
})
export class DurationMsPipe implements PipeTransform {
    transform(value: number): string {
        return DurationService.durationMs(value);
    }
}
