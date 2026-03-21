use bollard::Docker;
use futures_util::StreamExt;

#[tokio::main]
async fn main() {
    let docker = Docker::connect_with_unix(
        &std::env::var("DOCKER_HOST").unwrap(),
        120,
        &bollard::ClientVersion {
            major_version: 1,
            minor_version: 47,
        },
    )
    .unwrap();

    // List containers
    #[allow(deprecated)]
    let containers = docker
        .list_containers(None::<bollard::container::ListContainersOptions<String>>)
        .await
        .unwrap();
    println!("Found {} containers", containers.len());
    for c in &containers {
        let name = c.names.as_ref().and_then(|n| n.first()).unwrap();
        println!("  {} ({})", name, c.id.as_deref().unwrap_or("?"));
    }

    if containers.is_empty() {
        return;
    }

    let name = containers[0]
        .names
        .as_ref()
        .unwrap()
        .first()
        .unwrap()
        .trim_start_matches('/');
    println!("\nFetching logs for: {}", name);

    let opts = bollard::query_parameters::LogsOptionsBuilder::default()
        .follow(false)
        .stdout(true)
        .stderr(true)
        .tail("10")
        .until(i32::MAX) // Workaround: bollard serializes until=0 which filters all logs
        .build();

    let mut stream = docker.logs(name, Some(opts));
    let mut count = 0;

    while let Some(result) = stream.next().await {
        match result {
            Ok(output) => {
                count += 1;
                let bytes = output.into_bytes();
                let text = String::from_utf8_lossy(&bytes);
                println!("  [{}] {} bytes: {}", count, bytes.len(), text.trim());
            }
            Err(e) => {
                eprintln!("  Error: {}", e);
                break;
            }
        }
    }

    println!("Total log items: {}", count);
}
