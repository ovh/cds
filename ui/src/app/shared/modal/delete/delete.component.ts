import {Component, Output, EventEmitter, Input, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';

@Component({
    selector: 'app-delete-modal',
    templateUrl: './delete.html',
    styleUrls: ['./delete.scss']
})
export class DeleteModalComponent {

    private currentEvent: any;

    @Input() title: string;
    @Input() msg: string;
    @Output() event = new EventEmitter<any>();

    @ViewChild('myDeleteModal')
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
