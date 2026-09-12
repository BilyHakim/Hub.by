use tauri::{webview::WebviewWindowBuilder, WebviewUrl};

const DESKTOP_PORT: u16 = 1420;

fn main() {
    tauri::Builder::default()
        // A localhost origin keeps the existing cookie-based Go API authentication working.
        .plugin(tauri_plugin_localhost::Builder::new(DESKTOP_PORT).build())
        .setup(|app| {
            let app_url = if cfg!(debug_assertions) {
                "http://localhost:5173".to_owned()
            } else {
                format!("http://localhost:{DESKTOP_PORT}")
            };

            WebviewWindowBuilder::new(
                app,
                "main",
                WebviewUrl::External(app_url.parse()?),
            )
            .title("Hubby")
            .inner_size(1280.0, 800.0)
            .min_inner_size(960.0, 640.0)
            .build()?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running Hubby desktop");
}
