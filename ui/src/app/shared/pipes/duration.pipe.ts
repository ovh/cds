import { Pipe, PipeTransform } from '@angular/core';
import { DurationService } from '../../../../libs/workflow-graph/src/lib/duration.service';

@Pipe({
    name: 'durationMs',
    standalone: false
})
export class DurationMsPipe implements PipeTransform {
    transform(value: number | Date): string {
        if (value instanceof Date) {
            return DurationService.durationMs(Date.now() - value.getTime());
        }
        return DurationService.durationMs(value);
    }
}
