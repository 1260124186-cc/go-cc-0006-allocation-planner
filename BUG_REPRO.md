# 修复前故障复现（Docker）

## 项目与标准命令

库存分配计划器根据库存和订单行生成预留结果，并保留内存审计记录供汇总使用。在仓库根目录执行：

```sh
go build ./...
go run ./cmd/planner -action allocate < examples/request.json
go test ./...
```

## 环境构建与编译

linux/amd64：

```sh
docker buildx build --load --platform linux/amd64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner:amd64 .
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner:amd64 -c 'go build ./...'
```

linux/arm64：

```sh
docker buildx build --load --platform linux/arm64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner:arm64 .
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner:arm64 -c 'go build ./...'
```

两个平台的镜像构建和容器内编译均成功。目标故障在下节触发。

## 故障触发步骤

在修复前的源码状态下，于仓库根目录执行：

```sh
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner:arm64 -c 'go test ./internal/service -run TestAllocateHonorsCanceledContext -count=1'
```

## 实际错误输出

```text
--- FAIL: TestAllocateHonorsCanceledContext (0.08s)
    planner_test.go:66: Allocate() error = <nil>, want context deadline
FAIL
FAIL	github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service	0.084s
FAIL
```

## 期望行为

设置短超时的分配请求应尽快返回超时结果，不应返回成功；库存不应被扣减，后续汇总也不应出现新的分配记录。
