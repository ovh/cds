import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from "@angular/core";
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { dump} from "js-yaml";

declare const monaco: any;

@Component({
    standalone: false,
    selector: 'app-project-run-retention-report',
    templateUrl: './retention.report.html',
    styleUrls: ['./retention.report.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectRunRetentionReportComponent implements OnInit {


    @Input() report: any;
    @ViewChild('editor') editor: NzCodeEditorComponent;

    dataEditor: string;
    editorOption: EditorOptions;

    constructor(private _cd: ChangeDetectorRef) {
        this.editorOption = {
            language: 'yaml',
            minimap: { enabled: false }
        };
    }

    ngOnInit(): void {
        this.dataEditor = dump(this.report);
    }

    onEditorChange(event: string) {
        this.dataEditor = event;
        this._cd.markForCheck();
    }
}
