#!/bin/bash

set -e

git diff origin/main HEAD --name-only > diff.out
echo "_____________start__________"
cat diff.out
echo "_____________end__________"
test_folder="bpf/kmesh/bpf2go/bpf2go.go | pkg/utils/test/"
modify_go_files=$(grep -E "*\.go" diff.out)
if [ -z $modify_go_files ]; then
  echo "No modified go files is found."
else
  echo "start incrememtal UT check now..."
  go test -v -coverpkg=./... ./... -coverprofile=cover.out
  if [ "$?" != "0" ]; then
    echo "unit test failed"
    exit 1
  fi
  gocov convert cover.out | gocov-xml > coverage.xml
  diff_res=$(diff-cover coverage.xml --compare-branch=origin/main --html-report report.html)
  echo "diff_res: $diff_res"
  new_added=$(git diff origin/main HEAD --numstat | awk 'BEGIN {added=0} {added+=$1} END {print added}')
  if [[ $new_added -eq 0 ]]; then
    #  If the code is deleted, there is no need to add the UT
    echo "new_added equal to 0"
    coverage=100
  else
    total=$(echo "$diff_res" | grep Total | awk '{print int($2)}')
    echo "total: $total"
    if [[ $total -lt 20 ]]; then
      echo "less then 20"
      coverage=100
    else
      coverage=$(echo "$diff_res" | grep -i Coverage: | awk '{print $2}')
    fi
  fi
  echo "coverage: $coverage"
fi