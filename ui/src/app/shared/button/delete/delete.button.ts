import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
    selector: 'app-delete-button',
    templateUrl: './delete.html',
    styleUrls: ['./delete.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class DeleteButtonComponent {
    @Input() loading = false;

    // normal / icon
    @Input() buttonType = 'normal';
    @Input() disabled = false;
    @Input() class: string;
    @Input() title: string;
    @Output() event = new EventEmitter<boolean>();

    showConfirmation = false;

    constructor() { }

    deleteEvent() {
        this.event.emit(true);
        this.reset();
    }

    reset(): void {
        this.showConfirmation = false;
    }
}
