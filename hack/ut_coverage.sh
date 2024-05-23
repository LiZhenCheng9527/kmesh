#!/bin/bash

set -e

# 获取变更文件列表
MODIFIED_FILES=$(git diff --name-only origin/main...HEAD | grep '\.go$')
if [ -z "$MODIFIED_FILES" ]; then
  echo "No Go files changed. Skipping coverage check."
  exit 1
fi

# 运行测试并生成覆盖率报告
mkdir -p coverage
go test -coverprofile=coverage/coverage.out ./...

# 解析覆盖率报告，计算变更文件的覆盖率
TOTAL_COVERED=0
TOTAL_STMTS=0
while IFS= read -r FILE; do
  FILE_COVERAGE=$(go tool cover -func=coverage/coverage.out | grep "$FILE" | awk '{print $3}' | sed 's/%//')
  if [ -n "$FILE_COVERAGE" ]; then
    IFS='/' read -r COVERED STMTS <<< "$FILE_COVERAGE"
    TOTAL_COVERED=$((TOTAL_COVERED + COVERED))
    TOTAL_STMTS=$((TOTAL_STMTS + STMTS))
  fi
done <<< "$MODIFIED_FILES"

if [ $TOTAL_STMTS -eq 0 ]; then
  COVERAGE=100
else
  COVERAGE=$(echo "scale=2; ($TOTAL_COVERED / $TOTAL_STMTS) * 100" | bc)
fi

echo "Coverage for modified files: $COVERAGE%"

# 检查覆盖率是否达到预设阈值
THRESHOLD=80.0
if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
  echo "Coverage check failed! Current coverage: $COVERAGE%"
  exit 1
else
  echo "Coverage check passed! Current coverage: $COVERAGE%"
fi