import {Injectable} from '@angular/core';
import {Pipeline} from 'app/model/pipeline.model';
import {BehaviorSubject, Observable} from 'rxjs';

@Injectable()
export class PipelineCoreService {

    private _asCodeEditorEvent: BehaviorSubject<{open: boolean, save: boolean}> = new BehaviorSubject(null);
    private _previewPipeline: BehaviorSubject<Pipeline> = new BehaviorSubject(null);


    getAsCodeEditor(): Observable<{open: boolean, save: boolean}> {
        return new Observable<{open: boolean, save: boolean}>(fn => this._asCodeEditorEvent.subscribe(fn));
    }

    toggleAsCodeEditor(o: {open: boolean, save: boolean}): void {
        this._asCodeEditorEvent.next(o);
    }

    setPipelinePreview(pip: Pipeline): void {
        if (pip) {
            pip.forceRefresh = true;
            pip.previewMode = true;
        }
        this._previewPipeline.next(pip);
    }
}
