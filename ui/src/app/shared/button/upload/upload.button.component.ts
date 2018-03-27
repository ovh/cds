import {EventEmitter, Output, Input, Component} from '@angular/core';

@Component({
    selector: 'app-upload-button',
    templateUrl: './upload.button.html',
    styleUrls: ['./upload.button.scss']
})
export class UploadButtonComponent  {

    @Input() accept: string;

    @Input() size: string;
    @Output() event = new EventEmitter<string>();

    showConfirmation = false;

    constructor() {}

    fileEvent(event) {
      if (!event || !event.target || !event.target.files || !event.target.files[0]) {
        return;
      }
      let file = event.target.files[0];
      let reader = new FileReader();
      let that = this;

      reader.onloadend = function(e: any) {
        that.event.emit(e.target.result);
      };

      reader.readAsText(file);
    }
}
