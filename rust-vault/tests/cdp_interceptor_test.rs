//! TDD Tests for CDP Interceptor
//!
//! Tests verify that CDP interceptor filters by resourceType (XHR, Fetch only).

use rust_vault::blindfill::cdp_interceptor::CdpInterceptor;
use serde_json::json;

#[test]
fn test_enable_params_generates_correct_structure() {
    let params = CdpInterceptor::enable_params();

    assert_eq!(params["method"], "Fetch.enable");
    assert!(params["params"].is_object());

    let patterns = &params["params"]["patterns"];
    assert!(patterns.is_array());
    assert_eq!(patterns.as_array().unwrap().len(), 2);
}

#[test]
fn test_enable_params_filters_xhr_resource_type() {
    let params = CdpInterceptor::enable_params();
    let patterns = params["params"]["patterns"].as_array().unwrap();

    let xhr_pattern = patterns.iter().find(|p| p["resourceType"] == "XHR");
    assert!(xhr_pattern.is_some(), "XHR pattern not found");

    let xhr_pattern = xhr_pattern.unwrap();
    assert_eq!(xhr_pattern["requestStage"], "Request");
    assert_eq!(xhr_pattern["urlPattern"], "*");
}

#[test]
fn test_enable_params_filters_fetch_resource_type() {
    let params = CdpInterceptor::enable_params();
    let patterns = params["params"]["patterns"].as_array().unwrap();

    let fetch_pattern = patterns.iter().find(|p| p["resourceType"] == "Fetch");
    assert!(fetch_pattern.is_some(), "Fetch pattern not found");

    let fetch_pattern = fetch_pattern.unwrap();
    assert_eq!(fetch_pattern["requestStage"], "Request");
    assert_eq!(fetch_pattern["urlPattern"], "*");
}

#[test]
fn test_enable_params_only_includes_xhr_and_fetch() {
    let params = CdpInterceptor::enable_params();
    let patterns = params["params"]["patterns"].as_array().unwrap();

    let resource_types: Vec<&str> = patterns
        .iter()
        .map(|p| p["resourceType"].as_str().unwrap())
        .collect();

    assert_eq!(resource_types.len(), 2);
    assert!(resource_types.contains(&"XHR"));
    assert!(resource_types.contains(&"Fetch"));

    let forbidden_types = vec![
        "Document", "Stylesheet", "Image", "Media", "Font",
        "Script", "TextTrack", "EventSource", "WebSocket", "Manifest", "Other"
    ];

    for forbidden in forbidden_types {
        assert!(!resource_types.contains(&forbidden),
                "Forbidden resource type {} found in patterns", forbidden);
    }
}

#[test]
fn test_cdp_interceptor_struct_exists() {
    let _interceptor = CdpInterceptor::new();
}

#[test]
fn test_request_paused_handler_placeholder_resolution() {
    let interceptor = CdpInterceptor::new();
    let mut request_body = json!({
        "payment": "{{secret:payment.card_number}}"
    });

    let mut secrets = std::collections::HashMap::new();
    secrets.insert("payment.card_number".to_string(), "4242424242424242".to_string());

    let result = interceptor.resolve_placeholders(&mut request_body, &secrets);

    assert!(result.is_ok());
    assert_eq!(request_body["payment"], "4242424242424242");
}
