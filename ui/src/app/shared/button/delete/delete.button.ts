import {EventEmitter, Output, Input, Component} from '@angular/core';

@Component({
    selector: 'app-delete-button',
    templateUrl: './delete.html',
    styleUrls: ['./delete.scss']
})
export class DeleteButtonComponent  {

    @Input() loading = false;

    // normal / icon
    @Input() buttonType = 'normal';
    @Input() disabled = false;
    @Input() class: string;
    @Output() event = new EventEmitter<boolean>();

    showConfirmation = false;

    constructor() {}

    deleteEvent() {
        this.event.emit(true);
        this.reset();
    }

    reset(): void {
        this.showConfirmation = false;
    }
}
