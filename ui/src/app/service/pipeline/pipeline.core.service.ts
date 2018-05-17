import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Observable} from 'rxjs/Observable';
import {Pipeline} from '../../model/pipeline.model';

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

    getPipelinePreview(): Observable<Pipeline> {
        return new Observable<Pipeline>(fn => this._previewPipeline.subscribe(fn));
    }

    setPipelinePreview(pip: Pipeline): void {
        if (pip) {
            pip.forceRefresh = true;
            pip.previewMode = true;
        }
        this._previewPipeline.next(pip);
    }
}
