import { Pipe, PipeTransform } from '@angular/core';
import { DurationService } from '../../../../libs/workflow-graph/src/lib/duration.service';

@Pipe({
    name: 'durationMs',
    standalone: false
})
export class DurationMsPipe implements PipeTransform {
    transform(value: number): string {
        return DurationService.durationMs(value);
    }
}
