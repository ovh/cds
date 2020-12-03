import { SecurityContext } from '@angular/core';
import {ɵDomSanitizerImpl, DomSanitizer} from '@angular/platform-browser';
import {SafeHtmlPipe} from './safeHtml.pipe';

describe('Pipe: Default', () => {
  let pipe: SafeHtmlPipe;
  let sanitizer: DomSanitizer = new ɵDomSanitizerImpl(null);

  beforeEach(() => {
    pipe = new SafeHtmlPipe(sanitizer);
  });

  it('providing html value with script inside and sanitize', () => {
    const elt = pipe.transform(`<script>alert("test")</script> and <div onclick="alert('test')"></div>`, false);
    const sanitizedValue = sanitizer.sanitize(SecurityContext.HTML, elt);
    expect(sanitizedValue).toBe(' and <div></div>');
  });
});
