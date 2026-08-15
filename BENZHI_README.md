# 库存分配计划器

该命令行工具根据输入库存和订单行生成库存预留结果，并保留一条内存审计记录供汇总使用。项目不依赖网络或数据库，适合在本地或 Docker 容器内从源码重复构建和运行。

在仓库根目录执行：

```sh
go build ./...
go run ./cmd/planner -action allocate < examples/request.json
go test ./...
```

## 固定环境

Docker 文件为 `benzhi.Dockerfile`。`go.mod` 固定 Go 语言版本为 `1.26.0`，并指定 `go1.26.2` 工具链；Docker 镜像同样使用 `golang:1.26.2-alpine`，且通过 `GOTOOLCHAIN=local` 禁止容器在构建时自动下载其他工具链。镜像始终复制源码并在容器内执行 `go mod download` 和 `go build ./...`，不使用宿主机二进制。

可分别验收两个平台：

```sh
./build_benzhi_docker.sh linux/amd64
./build_benzhi_docker.sh linux/arm64
```

脚本会依次构建对应平台镜像、在容器内执行 `go build ./...`，再启动实际入口并读取镜像中的 `examples/request.json`。最后一个命令会输出一段 JSON 分配结果。

## 手工 Docker 命令

下面命令可逐步完成编译、运行和测试：

```sh
docker buildx build --load --platform linux/amd64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner:amd64 .
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner:amd64 -c 'go build ./...'
docker run --rm --platform linux/amd64 go-cc-0006-allocation-planner:amd64
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner:amd64 -c 'go test ./...'
```

将上述 `linux/amd64` 和镜像标签中的 `amd64` 同时替换为 `linux/arm64` 与 `arm64`，即可执行另一平台的同等验收。

通过标准是每个命令均以退出码 `0` 结束；容器内编译和测试均成功，运行命令输出包含订单标识和已分配的库存行的 JSON 结果。
