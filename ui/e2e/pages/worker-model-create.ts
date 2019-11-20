import { ClientFunction, Selector, t } from "testcafe";
import config from '../config';

export default class WorkerModelEditPage {

    inputName: Selector;
    groupName: Selector;
    type: Selector;
    dockerImage: Selector;
    patternScript: Selector;
    saveButton: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor() {
        this.inputName = Selector('#model-name');
        this.groupName = Selector('#group');
        this.type = Selector('div.ui.dropdown.type');
        this.dockerImage = Selector('#image');
        this.patternScript = Selector('#pattern');
        this.saveButton = Selector('#save-model-button');
    }

    async createDockerModel(name: string, group: string, image: string) {
        await t
            .typeText(this.inputName, name)
            .typeText(Selector('input.search').nth(0), group).pressKey('enter')
            .click(this.type).click(Selector('div.item[data-value=docker]'))
            .typeText(this.dockerImage, image)
            .click(this.patternScript).click(Selector('#pattern > div.menu > sui-select-option').nth(0))
            .click(this.saveButton)
            .expect(this.getLocation()).eql(config.baseUrl + '/settings/worker-model/'+group+'/'+name);
    }
}
