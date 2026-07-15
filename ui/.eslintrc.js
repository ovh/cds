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
            plugins: [
                "@typescript-eslint"
            ],
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
                ],
                 "@angular-eslint/prefer-standalone": [
					"off"
				],
                "@angular-eslint/prefer-inject": [
                    "off"
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
                ],
                "@angular-eslint/template/prefer-control-flow": [
                    "off"
                ],
                // Accessibility rules (see ui/docs/accessibility.md).
                // A rule runs at "warn" while legacy templates still trigger it
                // and is switched to "error" once the codebase is clean for it.
                "@angular-eslint/template/alt-text": ["warn"],
                "@angular-eslint/template/click-events-have-key-events": ["warn"],
                "@angular-eslint/template/elements-content": ["warn"],
                "@angular-eslint/template/interactive-supports-focus": ["warn"],
                "@angular-eslint/template/label-has-associated-control": ["warn"],
                "@angular-eslint/template/mouse-events-have-key-events": ["warn"],
                "@angular-eslint/template/no-autofocus": ["off"],
                "@angular-eslint/template/no-distracting-elements": ["warn"],
                "@angular-eslint/template/role-has-required-aria": ["warn"],
                "@angular-eslint/template/table-scope": ["warn"],
                "@angular-eslint/template/valid-aria": ["warn"]
            }
        }
    ]
};
