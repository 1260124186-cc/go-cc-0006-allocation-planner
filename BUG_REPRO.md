# 修复前故障复现（Docker）

## 项目与标准命令

该命令行工具根据输入库存和订单行生成库存预留结果，并保留内存审计记录供汇总使用。在修复前源码快照的仓库根目录可执行以下标准命令：

```sh
go build ./...
go run ./cmd/planner -action allocate < examples/request.json
go test ./...
```

## 环境构建与编译

已实际执行以下 linux/amd64 和 linux/arm64 镜像构建命令：

```sh
docker buildx build --load --platform linux/amd64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner-bug-005-base:amd64 .
docker buildx build --load --platform linux/arm64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner-bug-005-base:arm64 .
```

已在两个平台的容器内实际执行并成功完成编译：

```sh
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner-bug-005-base:amd64 -c 'go build ./...'
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner-bug-005-base:arm64 -c 'go build ./...'
```

两个平台的镜像构建和容器内编译均成功；目标故障在下节命令中触发。

## 故障触发步骤

在修复前源码快照的仓库根目录执行：

```sh
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner-bug-005-base:arm64 -c 'go test ./internal/service -run TestReportReleasesAuditSessionOnFailedSnapshot -count=1 -timeout=2s'
```

## 实际错误输出

```text
panic: test timed out after 2s
	running tests:
		TestReportReleasesAuditSessionOnFailedSnapshot (2s)

goroutine 33 [running]:
testing.(*M).startAlarm.func1()
	/usr/local/go/src/testing/testing.go:2802 +0x2c4
created by time.goFunc
	/usr/local/go/src/time/sleep.go:215 +0x38

goroutine 1 [chan receive]:
testing.(*T).Run(0x7ae90ecb8008, {0x193505?, 0x7ae90eca9b18?}, 0x1960f8)
	/usr/local/go/src/testing/testing.go:2109 +0x3bc
testing.runTests.func1(0x7ae90ecb8008)
	/usr/local/go/src/testing/testing.go:2585 +0x38
testing.tRunner(0x7ae90ecb8008, 0x7ae90eca9c48)
	/usr/local/go/src/testing/testing.go:2036 +0xc4
testing.runTests({0x1947c6, 0x38}, {0x195888, 0x49}, 0x7ae90eb82060, {0x2ce1e0, 0x4, 0x4}, {0x8069782c0434040a?, 0x18a70b?, ...})
	/usr/local/go/src/testing/testing.go:2583 +0x3f0
testing.(*M).Run(0x7ae90ebf4140)
	/usr/local/go/src/testing/testing.go:2443 +0x578
main.main()
	_testmain.go:54 +0x80

goroutine 7 [sync.Mutex.Lock]:
internal/sync.runtime_SemacquireMutex(0x2fa300?, 0x0?, 0x7ae90ebf0ca8?)
	/usr/local/go/src/runtime/sema.go:95 +0x28
internal/sync.(*Mutex).lockSlow(0x7ae90ebf8720)
	/usr/local/go/src/internal/sync/mutex.go:149 +0x170
internal/sync.(*Mutex).Lock(...)
	/usr/local/go/src/internal/sync/mutex.go:70
sync.(*Mutex).Lock(...)
	/usr/local/go/src/sync/mutex.go:46
github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/store.(*Audit).OpenReport(0x7ae90ebf8720, {0x1991b8?, 0x2fa300?})
	/workspace/internal/store/audit.go:39 +0x88
github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service.(*Planner).Report(0x7ae90ebf0dc0, {0x1991b8, 0x2fa300})
	/workspace/internal/service/report.go:10 +0x30
github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service_test.TestReportReleasesAuditSessionOnFailedSnapshot(0x7ae90ecb8248)
	/workspace/internal/service/planner_test.go:83 +0x254
testing.tRunner(0x7ae90ecb8248, 0x1960f8)
	/usr/local/go/src/testing/testing.go:2036 +0xc4
created by testing.(*T).Run in goroutine 1
	/usr/local/go/src/testing/testing.go:2101 +0x3a8
FAIL	github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service	2.008s
FAIL
```

## 期望行为

同一测试场景下，第一次报告读取失败应返回错误，后续报告请求应正常完成，不应持续阻塞或因超时失败。
