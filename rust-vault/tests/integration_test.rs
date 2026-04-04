use std::process::Command;
use std::path::Path;

#[tokio::test]
async fn test_project_compiles() {
    let output = Command::new("cargo")
        .args(&["check"])
        .current_dir(Path::new(env!("CARGO_MANIFEST_DIR")))
        .output()
        .expect("Failed to run cargo check");
    assert!(output.status.success(), "cargo check failed: {:?}", String::from_utf8_lossy(&output.stderr));
}
