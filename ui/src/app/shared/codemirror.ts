import { Component, Input, Output, ViewChild, EventEmitter, forwardRef, OnDestroy, AfterViewInit } from '@angular/core';
import { NG_VALUE_ACCESSOR } from '@angular/forms';

declare var CodeMirror: any;

@Component({
    // eslint-disable-next-line @angular-eslint/component-selector
    selector: 'codemirror',
    providers: [
        {
            provide: NG_VALUE_ACCESSOR,
            useExisting: forwardRef(() => CodemirrorComponent),
            multi: true
        }
    ],
    template: `<textarea [placeholder]="placeholder" #host></textarea>`,
})
export class CodemirrorComponent implements AfterViewInit, OnDestroy {
    @ViewChild('host', { static: false }) host;

    @Input() config;
    @Input() placeholder = '';
    @Output() change = new EventEmitter();
    @Output() instance = null;

    editor;
    _value = '';

    constructor() { }

    get value(): any {
        return this._value;
    };

    @Input() set value(v) {
        if (v !== this._value) {
            this._value = v;
            this.onChange(v);
        }
    }

    ngOnDestroy() { }

    ngAfterViewInit() {
        this.config = this.config || {};
        this.codemirrorInit(this.config);
    }

    codemirrorInit(config) {
        this.instance = CodeMirror.fromTextArea(this.host.nativeElement, config);
        this.instance.on('change', () => {
            this.updateValue(this.instance.getValue());
        });
    }

    updateValue(value) {
        this.value = value;
        this.onChange(value);
        this.onTouched();
        this.change.emit(value);
    }

    writeValue(value) {
        this._value = value || '';
        if (this.instance) {
            this.instance.setValue(this._value);
        }
    }

    onChange(_) { }

    onTouched() { }

    registerOnChange(fn) {
        this.onChange = fn;
    }

    registerOnTouched(fn) {
        this.onTouched = fn;
    }
}