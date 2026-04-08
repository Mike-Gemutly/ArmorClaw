fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=proto/governance.proto");
    tonic_build::compile_protos("proto/governance.proto")?;
    Ok(())
}
