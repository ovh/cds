import {
    AfterViewInit,
    ChangeDetectionStrategy,
    Component, ElementRef,
    EventEmitter,
    Input,
    Output,
    ViewChild
} from '@angular/core';

@Component({
    selector: 'app-input-autofocus',
    templateUrl: './autofocus.input.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AutoFocusInputComponent implements AfterViewInit {

    @Input() model: string;
    @Output() modelChange = new EventEmitter<string>();

    @ViewChild('searchinput') inp: ElementRef;

    ngAfterViewInit(): void {
        if (this.inp) {
            this.inp.nativeElement.focus();
        }
    }

    pushModification(): void {
        this.modelChange.emit(this.model);
    }
}
