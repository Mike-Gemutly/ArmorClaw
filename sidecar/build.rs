fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:warning=Proto generation skipped - grpc module disabled pending protoc installation");
    Ok(())
}
