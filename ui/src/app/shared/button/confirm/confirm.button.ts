import {Component, EventEmitter, Input, Output} from '@angular/core';

@Component({
    selector: 'app-confirm-button',
    templateUrl: './confirm.button.html',
    styleUrls: ['./confirm.button.scss']
})
export class ConfirmButtonComponent  {

    @Input() loading = false;
    @Input() icon = '';
    @Input() disabled = false;
    @Input() color = 'primary';
    @Input() class: string;
    @Input() title: string;
    @Output() event = new EventEmitter<boolean>();

    showConfirmation = false;

    constructor() {}

    confirmEvent() {
        this.event.emit(true);
        this.reset();
    }

    reset(): void {
        this.showConfirmation = false;
    }
}
