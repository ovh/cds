import { Component, HostListener } from '@angular/core';

@Component({
  selector: 'app-preview',
  templateUrl: './preview.component.html',
  styleUrls: ['./preview.component.scss']
})
export class PreviewComponent {
  fileContent: string = '';

  @HostListener('window:message', ['$event'])
  onRefresh(e: MessageEvent) {
    console.log(e);
    if (e?.data?.value) {
      this.fileContent = e.data.value;
    }
  }
}
