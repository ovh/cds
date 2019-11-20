import config from '../config';
import { ClientFunction, Selector, t } from "testcafe";

export default class WorkerModelPage {

    addPath: string;
    getLocation = ClientFunction(() => document.location.href);

    constructor() {
        this.addPath = '/settings/worker-model/add';
    }

    async clickAddButton() {
        await t.click(Selector('[href="'+this.addPath+'"]'))
            .expect(this.getLocation()) .eql(config.baseUrl + this.addPath);
    }
}
