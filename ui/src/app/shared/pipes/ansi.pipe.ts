import { Pipe, PipeTransform } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';
import * as AU from 'ansi_up';

@Pipe({ name: 'ansi' })
export class AnsiPipe implements PipeTransform {
    constructor(private sanitized: DomSanitizer) { }
    transform(value: string, disable: boolean): string {
        if (disable) {
            return value;
        }

        let ansiUp = new AU.default();
        return ansiUp.ansi_to_html(value);
    }
}
