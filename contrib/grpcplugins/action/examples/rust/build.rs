fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::compile_protos("../../../../../sdk/grpcplugin/actionplugin/actionplugin.proto")?;
    Ok(())
}
