import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { NzUploadFile } from 'ng-zorro-antd/upload';

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

    that: UploadButtonComponent

    constructor() {
        this.that = this;
    }

    fileEvent = (file: NzUploadFile): boolean => {
        const myReader = new FileReader();
        let that = this;
        myReader.onloadend = (e) => {
            // @ts-ignore
            this.event.emit({content: myReader.result, file: file as any})
        };

        if (this.image) {
            myReader.readAsDataURL(file as any);
        } else {
            myReader.readAsText(file as any);
        }
        return false;
    }
}
