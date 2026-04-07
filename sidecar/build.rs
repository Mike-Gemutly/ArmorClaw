use std::path::Path;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=src/grpc/proto/sidecar.proto");

    let generated_file = Path::new("src/grpc/proto/armorclaw.sidecar.v1.rs");

    if generated_file.exists() {
        println!("cargo:warning=Using pre-generated protobuf code");
        return Ok(());
    }

    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile(
            &["src/grpc/proto/sidecar.proto"],
            &["src/grpc/proto"],
        )?;

    println!("cargo:warning=Protobuf code generated successfully");
    Ok(())
}
