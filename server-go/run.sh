GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o mods/vanilla/vanilla.wasm mods/vanilla/main.go
go run .