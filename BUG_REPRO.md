# 修复前故障复现（Docker）

## 项目与标准命令

库存分配计划器根据输入库存和订单行生成库存预留结果，并保留内存审计记录供汇总使用。在仓库根目录可执行以下标准命令：

```sh
go build ./...
go run ./cmd/planner -action allocate < examples/request.json
go test ./...
```

## 环境构建与编译

已分别在 `linux/amd64` 和 `linux/arm64` 构建镜像，两个平台的镜像构建与容器内 `go build ./...` 均成功：

```sh
docker buildx build --load --platform linux/amd64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner-bug-004-base:amd64 .
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner-bug-004-base:amd64 -c 'go build ./...'
docker buildx build --load --platform linux/arm64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner-bug-004-base:arm64 .
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner-bug-004-base:arm64 -c 'go build ./...'
```

## 故障触发步骤

在仓库根目录执行以下命令，容器内目标测试会稳定失败：

```sh
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner-bug-004-base:arm64 -c 'go test ./internal/service -run TestAllocateReleasesStockWhenAuditFails -count=1'
```

## 实际错误输出

```text
--- FAIL: TestAllocateReleasesStockWhenAuditFails (0.00s)
    planner_test.go:50: available book = 0, want 2
FAIL
FAIL	github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service	0.002s
FAIL
退出状态：1
```

## 期望行为

当订单审计记录未能写入时，同一订单应对调用方返回失败，并且该次失败不应消耗任何可用库存；使用相同库存和订单再次提交时，仍应能够按原始可用数量处理。
