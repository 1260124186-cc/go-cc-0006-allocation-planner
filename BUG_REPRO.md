# 修复前故障复现（Docker）

## 项目与标准命令

该项目是一个根据库存和订单行生成库存预留结果的命令行工具，并保留内存审计记录。请在仓库根目录执行以下标准命令：

```sh
go build ./...
go run ./cmd/planner -action allocate < examples/request.json
go test ./...
```

## 环境构建与编译

已实际执行以下命令构建两个平台镜像，并在各自容器内完成 `go build ./...`：

```sh
docker buildx build --load --platform linux/amd64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner-bug-002-delivery:amd64 .
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner-bug-002-delivery:amd64 -c 'go build ./...'
docker buildx build --load --platform linux/arm64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner-bug-002-delivery:arm64 .
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner-bug-002-delivery:arm64 -c 'go build ./...'
```

两个平台的镜像构建和容器内编译均成功。目标故障在下节命令中触发。

## 故障触发步骤

在仓库根目录执行：

```sh
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner-bug-002-delivery:arm64 -c 'go test ./internal/service -run TestAllocateRejectsMissingReservation -count=1; status=$?; printf "EXIT_STATUS=%s\n" "$status"; exit "$status"'
```

## 实际错误输出

```text
--- FAIL: TestAllocateRejectsMissingReservation (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered, repanicked]
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x12926c]

goroutine 7 [running]:
testing.tRunner.func1.2({0x15c4c0, 0x2c8da0})
	/usr/local/go/src/testing/testing.go:1974 +0x1a0
testing.tRunner.func1()
	/usr/local/go/src/testing/testing.go:1977 +0x318
panic({0x15c4c0?, 0x2c8da0?})
	/usr/local/go/src/runtime/panic.go:860 +0x12c
github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service.(*Planner).Allocate(0x5d4a4fed2f20, {0x199508, 0x2fa300}, {{0x18864a, 0x7}, {0x5d4a4fe640d8, 0x1, 0x1}})
	/workspace/internal/service/planner.go:38 +0x1cc
github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service_test.TestAllocateRejectsMissingReservation(0x5d4a4ff9a248)
	/workspace/internal/service/planner_test.go:90 +0xc0
testing.tRunner(0x5d4a4ff9a248, 0x1963e0)
	/usr/local/go/src/testing/testing.go:2036 +0xc4
created by testing.(*T).Run in goroutine 1
	/usr/local/go/src/testing/testing.go:2101 +0x3a8
FAIL	github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service	0.005s
FAIL
EXIT_STATUS=1
```

## 期望行为

当订单包含库存中不存在的商品标识时，命令应返回可识别的库存缺失错误并以非零状态结束，而不应异常崩溃或输出运行时 panic 栈。
