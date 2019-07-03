import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
    selector: 'app-confirm-button',
    templateUrl: './confirm.button.html',
    styleUrls: ['./confirm.button.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConfirmButtonComponent  {
    @Input() loading = false;
    @Input() icon = '';
    @Input() disabled = false;
    @Input() color = 'primary';
    @Input() class: string;
    @Output() event = new EventEmitter();
    @Input() title: string;

    showConfirmation = false;

    constructor() {}

    confirmEvent() {
        this.event.emit();
        this.reset();
    }

    reset(): void {
        this.showConfirmation = false;
    }
}
