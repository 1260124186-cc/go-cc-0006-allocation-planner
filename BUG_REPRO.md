# 修复前故障复现（Docker）

## 项目与标准命令

库存分配计划器根据库存和订单行生成库存预留结果，并输出 JSON 分配记录。在仓库根目录可执行：

```sh
go build ./...
go run ./cmd/planner -action allocate < examples/request.json
go test ./...
```

## 环境构建与编译

已实际执行以下 linux/amd64 命令，镜像构建和容器内编译均成功：

```sh
docker buildx build --load --platform linux/amd64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner:amd64 .
docker run --rm --platform linux/amd64 --entrypoint sh go-cc-0006-allocation-planner:amd64 -c 'go build ./...'
```

已实际执行以下 linux/arm64 命令，镜像构建和容器内编译均成功：

```sh
docker buildx build --load --platform linux/arm64 -f benzhi.Dockerfile -t go-cc-0006-allocation-planner:arm64 .
docker run --rm --platform linux/arm64 --entrypoint sh go-cc-0006-allocation-planner:arm64 -c 'go build ./...'
```

## 故障触发步骤

在仓库根目录执行：

```sh
go test ./internal/service -run TestAllocateMergesDuplicateLines -count=1
```

## 实际错误输出

```text
--- FAIL: TestAllocateMergesDuplicateLines (0.00s)
    planner_test.go:29: allocation lines = []domain.Line{domain.Line{SKU:"book", Quantity:1}, domain.Line{SKU:"book", Quantity:2}}, want one merged line
FAIL
FAIL	github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service	1.139s
FAIL
退出状态: 1
```

## 期望行为

同一订单中重复的 SKU 应在输出分配结果中汇总为一条库存行，数量为这些重复行数量之和；没有重复 SKU 的订单应继续按原有库存行和数量输出。
