// Karma configuration file, see link for more information
// https://karma-runner.github.io/1.0/config/configuration-file.html

module.exports = function (config) {
    config.set({
        basePath: '',
        frameworks: ['jasmine', '@angular-devkit/build-angular'],
        plugins: [
            require('karma-jasmine'),
            require('karma-chrome-launcher'),
            require('karma-jasmine-html-reporter'),
            require('karma-coverage'),
            require('karma-junit-reporter'),
            require('@angular-devkit/build-angular/plugins/karma')
        ],
        files: [
            {pattern: './src/test.ts', watched: false},
            {pattern: './src/assets/**/*.png', watched: false, included: false, served: true},
            {pattern: './node_modules/jquery/dist/jquery.js', watch: false, included: true, served: true},
            {pattern: './node_modules/fomantic-ui/dist/semantic.js', watch: false, included: true, served: true},
            {pattern: './node_modules/codemirror/lib/codemirror.js', watch: false, included: true, served: true},
            {pattern: './node_modules/dragula/dist/dragula.js', watch: false, included: true, served: true}
        ],
        client: {
            clearContext: false // leave Jasmine Spec Runner output visible in browser
        },
        jasmineHtmlReporter: {
            suppressAll: true // removes the duplicated traces
        },
        junitReporter: {
            outputDir: 'tests', // results will be saved as $outputDir/$browserName.xml
            outputFile: 'results.xml', // if included, results will be saved as $outputDir/$browserName/$outputFile
            suite: 'testSuite'
        },
        coverageReporter: {
            dir: require('path').join(__dirname, 'coverage'),
            reporters: [
                { type: 'html' },
                { type: 'lcovonly', subdir: '.', file: 'lcov.info' },
            ]
        },
        reporters: ['progress', 'coverage', 'junit', 'kjhtml'],
        port: 9876,
        colors: true,
        logLevel: config.LOG_WARN,
        autoWatch: true,
        concurrency: 1,
        browsers: ['ChromeHeadless'],
        browserNoActivityTimeout: 80000,
        customLaunchers: {
            ChromeHeadless: {
                base: 'Chrome',
                flags: [
                    '--no-sandbox',
                    // See https://chromium.googlesource.com/chromium/src/+/lkgr/headless/README.md
                    '--headless',
                    '--disable-gpu',
                    // Without a remote debugging port, Google Chrome exits immediately.
                    ' --remote-debugging-port=9222',
                ]
            }
        },
        singleRun: false
    });
};
