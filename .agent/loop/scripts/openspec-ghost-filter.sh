#!/bin/bash
# openspec-ghost-filter.sh <change-name>
# 判断 openspec/changes/<change-name> 是否已实际合并进代码(幽灵)
# exit 1 = 幽灵(应跳过), exit 0 = 真正待办
set -u
CHANGE="${1:?usage: $0 <change-name>}"
DIR="openspec/changes/$CHANGE"
[ -d "$DIR" ] || { echo "GHOST: dir not found" >&2; exit 0; }
# 只用 change 目录名里的 primary CYB tag,避免 cross-ref 误判
PRIMARY=$(echo "$CHANGE" | grep -oE '^CYB-[0-9]+' | head -1)
[ -z "$PRIMARY" ] && { echo "REAL: no CYB prefix in change name" >&2; exit 0; }
for T in $PRIMARY; do
  HITS=$(git log --all --oneline -i --grep="$T\b" 2>/dev/null | wc -l | tr -d ' ')
  if [ "$HITS" -gt 0 ]; then
    echo "GHOST: $CHANGE matches $T in $HITS commits" >&2
    exit 1
  fi
done
echo "REAL: $CHANGE tags $TAGS not in git log" >&2
exit 0
