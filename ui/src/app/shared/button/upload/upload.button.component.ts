import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
    selector: 'app-upload-button',
    templateUrl: './upload.button.html',
    styleUrls: ['./upload.button.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UploadButtonComponent  {

    @Input() accept: string;
    @Input() image: boolean;

    @Input() size: string;
    @Output() event = new EventEmitter<{content: string, file: File}>();

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
        that.event.emit({content: e.target.result, file});
      };

      if (this.image) {
        reader.readAsDataURL(file);
      } else {
        reader.readAsText(file);
      }
    }
}
