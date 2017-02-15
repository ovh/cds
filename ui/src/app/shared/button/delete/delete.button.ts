import {Component, } from '@angular/core/src/metadata/directives';
import {EventEmitter, Output, Input} from '@angular/core';

@Component({
    selector: 'app-delete-button',
    templateUrl: './delete.html',
    styleUrls: ['./delete.scss']
})
export class DeleteButtonComponent  {

    @Input() loading = false;
    @Output() event = new EventEmitter<boolean>();

    showConfirmation = false;

    constructor() {}

    deleteEvent() {
        this.event.emit(true);
    }

    reset(): void {
        this.showConfirmation = false;
    }
}
