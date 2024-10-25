module.exports = {
    root: true,
    ignorePatterns: [
        "projects/**/*",
        "extra-webpack.config.js"
    ],
    overrides: [
        {
            files: [
                "*.ts"
            ],
            parserOptions: {
                tsconfigRootDir: __dirname,
                project: [
                    "tsconfig.json"
                ],
                createDefaultProgram: true
            },
            extends: [
                "plugin:@angular-eslint/recommended",
                "plugin:@angular-eslint/template/process-inline-templates"
            ],
            rules: {
                "@angular-eslint/no-empty-lifecycle-method": [
                    "off"
                ],
                "@angular-eslint/no-output-on-prefix": [
                    "off"
                ],
                "@angular-eslint/no-output-native": [
                    "off"
                ],
                "@angular-eslint/directive-selector": [
                    "error",
                    {
                        "type": "attribute",
                        "prefix": "app",
                        "style": "camelCase"
                    }
                ],
                "@angular-eslint/component-selector": [
                    "error",
                    {
                        "type": "element",
                        "prefix": "app",
                        "style": "kebab-case"
                    }
                ]
            }
        },
        {
            "files": [
                "*.html"
            ],
            "extends": [
                "plugin:@angular-eslint/template/recommended"
            ],
            "rules": {
                "@angular-eslint/template/eqeqeq": [
                    "error",
                    {
                        "allowNullOrUndefined": true
                    }
                ]
            }
        }
    ]
};
