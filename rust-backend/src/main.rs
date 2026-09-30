use axum::{Router, routing::get};

#[tokio::main]
async fn main() {
    let router = Router::new().route("/health", get(health));

    let listener = tokio::net::TcpListener::bind("0.0.0.0:3000").await.unwrap();

    println!("App is listening on port 3000");
    axum::serve(listener, router).await.unwrap();
}

async fn health() -> &'static str {
    "ok"
}
