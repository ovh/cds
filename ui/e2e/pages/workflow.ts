import config from '../config';
import { Selector, t } from "testcafe";
import WorkflowTriggerFormPage from './workflow-trigger-form';

const workflowTriggerForm = new WorkflowTriggerFormPage();

export default class WorkflowPage {

    url: string;
    workflowTab: Selector;
    runWorkflowButton: Selector;

    // Modal
    modalRunWorkflowButton: Selector;

    constructor(key: string, workflowName: string) {
        this.url = config.baseUrl + '/project/' + key + '/workflow/' + workflowName;
        this.workflowTab = Selector('#WorkflowGraphTabs');
        this.runWorkflowButton = Selector('.ui.green.buttons>.ui.button');
        this.modalRunWorkflowButton = Selector('sui-modal-dimmer .ui.green.button');
    }

    async runWorkflow() {
        await t
            .click(this.runWorkflowButton)
            .click(this.modalRunWorkflowButton);
    }

    async addFork(cssClass: string) {
        await t
            .click(Selector(cssClass))
            .click(Selector('a.fork.item'))
    }

    async addPipeline(cssClass: string, index: number, pipName: string) {
        await t
            .click(Selector(cssClass).nth(index))
            .click(Selector('a.pipeline.item'));
        await workflowTriggerForm.addExistingPipeline(pipName);
    }

    async addJoin(cssClass: string, index: number) {
        await t
            .click(Selector(cssClass).nth(index))
            .click(Selector('a.add.join.item'));
    }

    async linkJoin(pipelineClass: string, index: number, joinIndex: number) {
        await t
            .click(Selector(pipelineClass).nth(index))
            .click(Selector('a.link.join.item'))
            .click(Selector('.node.workflowJoin.pointing > svg').nth(0))
    }

}
