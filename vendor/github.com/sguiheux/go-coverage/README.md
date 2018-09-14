# GO Coverage

go-coverage is a lcov/cobertura parser

## Usage

```
lcovParser := New("./test/lcov.info", LCOV)
report, err := lcovParser.Parse()
```