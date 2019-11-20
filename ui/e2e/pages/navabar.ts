import { ClientFunction, Selector, t } from 'testcafe';
import config from '../config';

export default class NavbarPage {

    createProjectLink: Selector;
    optionsDropdown: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor() {
        this.createProjectLink = Selector('[href="/project"]');
        this.optionsDropdown = Selector('div.ui.options.dropdown');
    }

    async clickCreateProject() {
        await t
            .click(this.createProjectLink)
            .expect(this.getLocation()).eql(config.baseUrl + '/project');
    }

    async clickWorkerModel() {
        await t
            .click(this.optionsDropdown)
            .click(Selector('[href="/settings/worker-model"]'))
            .expect(this.getLocation()).eql(config.baseUrl + '/settings/worker-model');
    }
}
