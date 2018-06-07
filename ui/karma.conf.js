// Karma configuration file, see link for more information
// https://karma-runner.github.io/0.13/config/configuration-file.html

module.exports = function (config) {
    config.set({
        basePath: '',
        frameworks: ['jasmine', '@angular-devkit/build-angular'],
        plugins: [
            require('karma-jasmine'),
            require('karma-chrome-launcher'),
            require('karma-coverage-istanbul-reporter'),
            require('karma-junit-reporter'),
            require('@angular-devkit/build-angular/plugins/karma')
        ],
        client:{
            clearContext: false // leave Jasmine Spec Runner output visible in browser
        },
        files: [
            {pattern: './src/test.ts', watched: false},
            {pattern: './src/assets/**/*.png', watched: false, included: false, served: true},
            {pattern: './node_modules/lodash/lodash.js', watch: false, included: true, served: true},
            {pattern: './node_modules/jquery/dist/jquery.js', watch: false, included: true, served: true},
            {pattern: './node_modules/semantic-ui/dist/semantic.js', watch: false, included: true, served: true},
            {pattern: './node_modules/codemirror/lib/codemirror.js', watch: false, included: true, served: true},
            {pattern: './node_modules/dragula/dist/dragula.js', watch: false, included: true, served: true}
        ],
        preprocessors: {
            
        },
        mime: {
            'text/x-typescript': ['ts', 'tsx']
        },
        coverageIstanbulReporter: {
            dir: require('path').join(__dirname, 'coverage'), reports: [ 'html', 'lcovonly' ],
            fixWebpackSourcePaths: true
        },
        angularCli: {
            environment: 'test'
        },
        reporters: config.angularCli && config.angularCli.codeCoverage
            ? ['progress', 'coverage-istanbul', 'junit']
            : ['progress'],
        junitReporter: {
            outputDir: 'tests', // results will be saved as $outputDir/$browserName.xml
            outputFile: 'results.xml', // if included, results will be saved as $outputDir/$browserName/$outputFile
            suite: 'testSuite'
        },
        port: 9876,
        colors: true,
        logLevel: config.LOG_INFO,
        autoWatch: true,
        browsers: ['ChromeHeadless'],
        browserNoActivityTimeout: 60000,
        singleRun: false,
        phantomjsLauncher: {
            exitOnResourceError: false
        },
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
    });
};
