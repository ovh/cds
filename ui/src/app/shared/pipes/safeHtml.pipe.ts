import { Pipe, PipeTransform } from '@angular/core';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import sanitizeHtml from 'sanitize-html';

@Pipe({ name: 'safeHtml' })
export class SafeHtmlPipe implements PipeTransform {
    constructor(private sanitized: DomSanitizer) { }
    transform(value: string): SafeHtml {
        let config = {
            allowedTags: ['b', 'font', 'i', 'em', 'strong', 'h1', 'h2', 'h3', 'h4', 'div', 'span'],
            allowedAttributes: {
                font: ['color'],
                span: ['style'],
            }
        };

        return this.sanitized.bypassSecurityTrustHtml(sanitizeHtml(value, config));
    }
}
