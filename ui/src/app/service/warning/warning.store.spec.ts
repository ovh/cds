/* tslint:disable:no-unused-variable */

import {TestBed, async} from '@angular/core/testing';
import {WarningStore} from './warning.store';
import {WarningAPI} from '../../model/warning.model';
import {Project} from '../../model/project.model';
import {Application} from '../../model/application.model';
import {Pipeline} from '../../model/pipeline.model';
import {Action} from '../../model/action.model';

describe('CDS: Warning Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
        });

    });

    it('Test Update Warning', async( () => {
        let warnStore: WarningStore = new WarningStore();
        warnStore.updateWarnings(getWarningFromAPI());

        let haveWarnings = false;
        warnStore.getWarnings().subscribe( res => {
            expect(res.get('key1')).toBeTruthy('Must have warnings on project key1');
            expect(res.get('key1').applications.get('app1')).toBeTruthy('Must have warnings on application app1');
            expect(res.get('key1').applications.get('app1').actions.length).toBe(3, 'App1 must have 3 warnings on jobs');

            expect(res.get('key1').pipelines.get('pip1')).toBeTruthy('Must have warnings on pipeline pip1');
            expect(res.get('key1').pipelines.get('pip1').jobs.length).toBe(1, 'Pip1 must have 1 warning on jobs');

            expect(res.get('key1').pipelines.get('pip2')).toBeTruthy('Must have warnings on pipeline pip2');
            expect(res.get('key1').pipelines.get('pip2').jobs.length).toBe(2, 'Pip2 must have 2 warning on jobs');


            expect(res.get('key2')).toBeTruthy('Must have warnings on project key2');
            expect(res.get('key2').applications.get('app2')).toBeTruthy('Must have warnings on application app2');
            expect(res.get('key2').applications.get('app2').actions.length).toBe(1, 'App2 must have 1 warning on actions');

            expect(res.get('key2').pipelines.get('pip3')).toBeTruthy('Must have warnings on pipeline pip2');
            expect(res.get('key2').pipelines.get('pip3').jobs.length).toBe(2, 'Pip3 must have 2 warnings on jobs');

            haveWarnings = true;
        });

        expect(haveWarnings).toBeTruthy('Warning store must have warning');

    }));

    function getWarningFromAPI(): WarningAPI[] {
        let warningsAPI: WarningAPI[] = [];
        warningsAPI.push(createWarning('key1', 'app1', 'pip1', 'action1', 1));

        warningsAPI.push(createWarning('key1', 'app1', 'pip2', 'action1', 1));
        warningsAPI.push(createWarning('key1', 'app1', 'pip2', 'action2', 1));


        warningsAPI.push(createWarning('key2', null, 'pip3', 'action3', 1));
        warningsAPI.push(createWarning('key2', 'app2', 'pip3', 'action4', 1));

        return warningsAPI;
    }

    function createWarning(key: string, appName: string, pipName: string, actionName: string, stageID: number) {
        let warn = new WarningAPI();

        let project: Project = new Project();
        project.key = key;

        let application = null;
        if (appName) {
            application = new Application();
            application.name = appName;
        }

        let pipeline = null;
        if (pipName) {
            pipeline = new Pipeline();
            pipeline.name = pipName;
        }

        let action = null;
        if (actionName) {
            action = new Action();
            action.name = actionName;
        }

        warn.action = action;
        warn.application = application;
        warn.pipeline = pipeline;
        warn.project = project;
        warn.stage_id = stageID !== 0 ? stageID : null;

        return warn;
    }
});
