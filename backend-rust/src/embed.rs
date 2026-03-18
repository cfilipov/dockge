use axum::{
    body::Body,
    extract::Request,
    http::{header, StatusCode},
    response::{IntoResponse, Response},
};
use rust_embed::Embed;
use std::path::Path;

#[derive(Embed)]
#[folder = "../dist/"]
struct FrontendAssets;

/// Serve embedded frontend assets with SPA fallback to index.html.
pub async fn spa_handler(req: Request) -> Response {
    let path = req.uri().path().trim_start_matches('/');

    // Try exact path first
    if let Some(resp) = serve_asset(path) {
        return resp;
    }

    // SPA fallback: serve index.html for client-side routing
    if let Some(resp) = serve_asset("index.html") {
        return resp;
    }

    StatusCode::NOT_FOUND.into_response()
}

fn serve_asset(path: &str) -> Option<Response> {
    let file = FrontendAssets::get(path)?;

    let mime = mime_guess::from_path(path)
        .first_raw()
        .unwrap_or("application/octet-stream");

    let body = Body::from(file.data.to_vec());

    Some(
        Response::builder()
            .status(StatusCode::OK)
            .header(header::CONTENT_TYPE, mime)
            .header(header::CACHE_CONTROL, cache_control(path))
            .body(body)
            .unwrap(),
    )
}

fn cache_control(path: &str) -> &'static str {
    // Hashed assets get long-term caching
    if let Some(ext) = Path::new(path).extension().and_then(|e| e.to_str())
        && path.starts_with("assets/")
        && matches!(ext, "js" | "css" | "woff2" | "woff" | "ttf")
    {
        return "public, max-age=31536000, immutable";
    }
    "public, max-age=0, must-revalidate"
}
