import { Component, HostListener } from '@angular/core';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {
  title = 'cds.workflow.preview';

  fileContent: string = '';

  @HostListener('window:message', ['$event'])
  onRefresh(e: MessageEvent) {
    if (e?.data?.value) {
      this.fileContent = e.data.value;
    }
  }
}
