import { StatusBarAlignment, StatusBarItem, window } from "vscode";
import { Property } from "./util.property";
import { Spinner } from "./view.spinner";
import { Journal } from "./util.journal";

export class StatusBarView {
    public static getInstance(): StatusBarView {
        if (!this.instance) {
            this.instance = new StatusBarView();
        }

        return this.instance;
    }

    private static instance: StatusBarView;
    private readonly statusBarItem: StatusBarItem;
    private progressInterval: NodeJS.Timer | undefined;
    private readonly spinner: Spinner;
    private spinnerActive: boolean = false;

    private constructor() {
        this.statusBarItem = window.createStatusBarItem(
            StatusBarAlignment.Left,
            Property.get("statusBarPositionPriority"),
        );
        this.spinner = new Spinner();
    }

    public clear(): void {
        this.stopProgress();
        this.setText("", false);
    }

    public stopProgress(): void {
        if (typeof this.progressInterval !== "undefined") {
            clearInterval(this.progressInterval);
            this.spinnerActive = false;
        }
    }

    public startProgress(): void {
        if (this.spinnerActive) {
            return;
        }

        this.stopProgress();

        if (this.spinner.updatable()) {
            this.progressInterval = setInterval(() => {
                this.setSpinner();
            }, 100);
        } else {
            this.setSpinner();
        }

        this.spinnerActive = true;
    }

    public dispose(): void {
        this.stopProgress();
        this.statusBarItem.dispose();
    }

    public update(proj: any, wrkflw: any): void {
        this.setText(`$(beaker) ${proj.key}/${wrkflw.name}`);
    }

    private setText(text: string, hasCommand: boolean = true): void {
        this.statusBarItem.text = text.trim();
        this.statusBarItem.tooltip = "CDS Workflow";
        this.statusBarItem.command = "";
        if (hasCommand) {
            this.statusBarItem.command = "extension.vsCdsOpenBrowserWorkflowStatusBar";
        }
        this.statusBarItem.show();
    }

    private setSpinner(): void {
        this.setText(`${this.spinner} Waiting for cdsctl response`, false);
    }
}
