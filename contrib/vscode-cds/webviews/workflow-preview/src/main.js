"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const core_1 = require("@angular/core");
const platform_browser_dynamic_1 = require("@angular/platform-browser-dynamic");
const app_module_1 = require("./app/app.module");
const environment_1 = require("./environments/environment");
if (environment_1.environment.production) {
    (0, core_1.enableProdMode)();
}
(0, platform_browser_dynamic_1.platformBrowserDynamic)().bootstrapModule(app_module_1.AppModule)
    .catch(err => console.error(err));
//# sourceMappingURL=main.js.map