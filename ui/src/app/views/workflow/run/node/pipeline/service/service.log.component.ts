import {Component, ElementRef, Input, OnInit, ViewChild} from '@angular/core';
import {ServiceLog} from '../../../../../../model/pipeline.model';
import {AutoUnsubscribe} from '../../../../../../shared/decorator/autoUnsubscribe';

declare var ansi_up: any;

@Component({
    selector: 'app-workflow-service-log',
    templateUrl: './service.log.html',
    styleUrls: ['service.log.scss']
})
@AutoUnsubscribe()
export class WorkflowServiceLogComponent implements OnInit {

    @Input() serviceLog: ServiceLog;

    @ViewChild('logsContent') logsElt: ElementRef;

    logsSplitted: Array<string> = [];

    showLog = false;

    constructor() { }

    ngOnInit(): void {
        this.logsSplitted = this.getLogs().split('\n');
    }

    getLogs() {
        if (this.serviceLog && this.serviceLog.val) {
            return ansi_up.ansi_to_html(this.serviceLog.val);
        }
        return '';
    }

    copyRawLog() {
        this.logsElt.nativeElement.value = this.serviceLog.val;
        this.logsElt.nativeElement.select();
        document.execCommand('copy');
    }
}
