import { CommonModule } from '@angular/common';
import { Component, HostListener } from '@angular/core';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  
  fileValue: String;

  constructor() {
    this.fileValue = '';
  }

  @HostListener('window:message', ['$event'])
  onRefresh(e: MessageEvent) {
    console.log(e);
    if (e.data.type === 'refresh') {
      this.fileValue = e.data.value;
    }
  }
}
