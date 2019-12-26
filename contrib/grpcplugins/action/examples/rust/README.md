# Rust plugin example

## How to build

```bash
cargo build --release
```

## Create plugin

```bash
cdsctl admin plugins import rust.yml
```

## Add new binary for this plugin

```bash
cdsctl admin plugins binary-add plugin-rust plugin.yml target/release/rust
```

## More informations

For more informations about how to develop and install plugins on CDS please check [our documentation](https://ovh.github.io/cds/development/contribute/plugin/).