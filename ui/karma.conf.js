// Karma configuration file, see link for more information
// https://karma-runner.github.io/0.13/config/configuration-file.html

module.exports = function (config) {
    config.set({
        basePath: '',
        frameworks: ['jasmine', '@angular/cli'],
        plugins: [
            require('karma-jasmine'),
            require('karma-phantomjs-launcher'),
            require('karma-chrome-launcher'),
            require('karma-remap-istanbul'),
            require('karma-junit-reporter'),
            require('@angular/cli/plugins/karma')
        ],
        files: [
            {pattern: './src/test.ts', watched: false},
            {pattern: './src/assets/**/*.png', watched: false, included: false, served: true},
            {pattern: './node_modules/lodash/lodash.js', watch: false, included: true, served: true},
            {pattern: './node_modules/jquery/dist/jquery.js', watch: false, included: true, served: true},
            {pattern: './node_modules/semantic-ui/dist/semantic.js', watch: false, included: true, served: true},
            {pattern: './node_modules/codemirror/lib/codemirror.js', watch: false, included: true, served: true}
        ],
        preprocessors: {
            './src/test.ts': ['@angular/cli']
        },
        mime: {
            'text/x-typescript': ['ts', 'tsx']
        },
        remapIstanbulReporter: {
            reports: {
                html: 'coverage',
                lcovonly: './coverage/coverage.lcov'
            }
        },
        angularCli: {
            config: './angular-cli.json',
            environment: 'source'
        },
        reporters: config.angularCli && config.angularCli.codeCoverage
            ? ['progress', 'karma-remap-istanbul', 'junit']
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
        //browsers: ['Chrome'],
        browsers: ['PhantomJS'],
        browserNoActivityTimeout: 60000,
        singleRun: false,
        phantomjsLauncher: {
            exitOnResourceError: false
        }
    });
};
