// Forked code from https://github.com/1-2-3/zorro-sharper

import {
  Directive,
  ElementRef,
  Input,
  SimpleChange,
  HostListener,
  ChangeDetectorRef,
} from '@angular/core';
import { NzTableComponent } from 'ng-zorro-antd/table';

@Directive({
  // eslint-disable-next-line @angular-eslint/directive-selector
  selector: '[nsAutoHeightTable]',
  exportAs: 'nsAutoHeightTable'
})
export class NsAutoHeightTableDirective {
  @Input('nsAutoHeightTable') offset: number = 0;

  constructor(
    private element: ElementRef,
    private table: NzTableComponent<any>,
    private cd: ChangeDetectorRef
  ) { }

  @HostListener('window:resize', ['$event'])
  onResize(event: any) {
    this.doAutoSize();
  }

  ngAfterViewInit() {
    this.doAutoSize();
  }

  private doAutoSize() {
    setTimeout(() => {
      if (!this.element?.nativeElement?.parentElement?.offsetHeight) {
        return;
      }
      const offset = this.offset || 0;
      const originNzScroll = this.table && this.table.nzScroll ? { ...this.table.nzScroll } : {};
      this.table.nzScroll = {
        ...originNzScroll,
        y: (this.element.nativeElement.parentElement.offsetHeight - offset).toString() + 'px'
      }
      this.table.ngOnChanges({
        nzScroll: new SimpleChange({ originNzScroll }, this.table.nzScroll, false),
      });
      this.cd.detectChanges();
    }, 10);
  }
}