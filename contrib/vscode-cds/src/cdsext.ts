import { CDSContext } from "./explorer";
import { StatusBarView } from "./view.statusbar";
import { Journal } from "./util.journal";
import { throttleFunction } from "./util.throttle.function";
import { workspace, window, Disposable } from "vscode";

export class CDSExt {
    public currentContext: CDSContext | undefined;
    public readonly statusBarView: StatusBarView = StatusBarView.getInstance();

    private static instance: CDSExt;
    private disposable: Disposable;

    public static getInstance(): CDSExt {
        if (!this.instance) {
            this.instance = new CDSExt();
        }
        return this.instance;
    }

    private constructor() {
        this.disposable = this.setupDisposables();
        this.setupListeners();
    }

    public dispose(): void {
        this.disposable.dispose();
    }

    private setupDisposables(): Disposable {
        const errorHandler = Journal.getInstance();
        return Disposable.from(this.statusBarView, errorHandler);
    }

    private setupListeners(): void {
        const disposables: Disposable[] = [];
        window.onDidChangeActiveTextEditor(this.onTextEditorMove, this, disposables);
        window.onDidChangeTextEditorSelection(this.onTextEditorMove, this, disposables);
        workspace.onDidSaveTextDocument(this.onTextEditorMove, this, disposables);
        this.disposable = Disposable.from(this.disposable, ...disposables);
    }

    @throttleFunction(5000)
    private async onTextEditorMove(): Promise<void> {
        // Only update if we haven't moved since we started blaming
        this.updateStatusBarView();
    }

    private async updateStatusBarView(): Promise<void> {
        this.statusBarView.startProgress();
        try {
            const proj = await this.currentContext!.cdsctl.getCDSProject();
            const wrkflw = await this.currentContext!.cdsctl.getCDSWorkflow();
            this.statusBarView.update(proj, wrkflw);
        } catch (e) {
            Journal.logError(e);
            this.statusBarView.stopProgress();
            this.clearStatusBarView();
        }
    }

    private clearStatusBarView() {
        this.statusBarView.clear();
    }
}
