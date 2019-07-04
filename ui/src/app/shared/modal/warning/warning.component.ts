import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { SemanticModalComponent } from 'ng-semantic/ng-semantic';

@Component({
    selector: 'app-warning-modal',
    templateUrl: './warning.html',
    styleUrls: ['./warning.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WarningModalComponent {

    private currentEvent: any;

    @Input() title: string;
    @Input() msg: string;
    @Output() event = new EventEmitter<any>();

    @ViewChild('myModal', {static: false})
    private modal: SemanticModalComponent;

    constructor() { }

    show(event?: any) {
        if (event) {
            this.currentEvent = event;
        }
        this.modal.show();
    }

    eventAndClose(modal: any) {
        this.event.emit(this.currentEvent);
        this.close(modal);
    }

    close(modal: any) {
        modal.hide();
    }
}
