set -e

for dir in api/*/; do
    [ -d "$dir" ] || continue
    service=$(basename "$dir")

    if [ ! -d "pkg/gen/$service" ]; then
        mkdir -p "pkg/gen/$service"
    fi
    protoc --go_out=pkg/gen/"$service" --go-grpc_out=pkg/gen/"$service" api/"$service"/*.proto
done