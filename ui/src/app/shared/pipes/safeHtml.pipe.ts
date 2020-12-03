import {Pipe, PipeTransform} from '@angular/core';
import {DomSanitizer, SafeHtml} from '@angular/platform-browser';
declare let sanitizeHtml: any;

@Pipe({ name: 'safeHtml'})
export class SafeHtmlPipe implements PipeTransform  {
  constructor(private sanitized: DomSanitizer) {}
  transform(value: string, trustHTML: boolean): SafeHtml {
    if (trustHTML) {
      return this.sanitized.bypassSecurityTrustHtml(value);
    }

    let config = {
      allowedTags: [ 'b', 'font', 'i', 'em', 'strong', 'h1', 'h2', 'h3', 'h4', 'div', 'span'],
      allowedAttributes: {
        font: ['color'],
        span: ['style'],
      }
    };

    return this.sanitized.bypassSecurityTrustHtml(sanitizeHtml(value, config));
  }
}
