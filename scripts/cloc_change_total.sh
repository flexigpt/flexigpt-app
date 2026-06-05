#!/usr/bin/env bash

ROOT="${1:-.}"
DAYS=15
EMPTY_TREE="4b825dc642cb6eb9a060e54bf8d69288fbee4904"

mapfile -t repos < <(
  find "$ROOT" \( -type d -name .git -prune -o -type f -name .git \) -print |
  sed 's#/.git$##' |
  sort
)

printf '%s ' "${repos[@]}"
echo

printf "%-12s %10s %10s %10s %12s %14s %8s\n" \
  "date" "added" "modified" "removed" "net_change" "change_total" "repos"



grand_added=0
grand_modified=0
grand_removed=0
grand_net=0
grand_change=0

for i in $(seq $((DAYS - 1)) -1 0); do
  day=$(date -d "$i days ago" +%F)
  next=$(date -d "$day +1 day" +%F)

  day_added=0
  day_modified=0
  day_removed=0
  changed_repos=0

  for repo in "${repos[@]}"; do
    start=$(git -C "$repo" rev-list -1 --before="$day 00:00" HEAD 2>/dev/null)
    end=$(git -C "$repo" rev-list -1 --before="$next 00:00" HEAD 2>/dev/null)

    [ -z "$end" ] && continue
    [ -z "$start" ] && start="$EMPTY_TREE"
    [ "$start" = "$end" ] && continue

    read added modified removed < <(
      cd "$repo" &&
      cloc --quiet \
        --exclude-lang=YAML \
        --git --diff "$start" "$end" 2>/dev/null |
      awk '
        /^SUM:/ { in_sum=1; next }
        in_sum && $1=="added"    { added=$5 }
        in_sum && $1=="modified" { modified=$5 }
        in_sum && $1=="removed"  { removed=$5 }
        END { printf "%d %d %d\n", added+0, modified+0, removed+0 }
      '
    )

    if [ "$added" -ne 0 ] || [ "$modified" -ne 0 ] || [ "$removed" -ne 0 ]; then
      changed_repos=$((changed_repos + 1))
    fi

    day_added=$((day_added + added))
    day_modified=$((day_modified + modified))
    day_removed=$((day_removed + removed))
  done

  net_change=$((day_added + day_modified - day_removed))
  change_total=$((day_added + day_modified + day_removed))

  grand_added=$((grand_added + day_added))
  grand_modified=$((grand_modified + day_modified))
  grand_removed=$((grand_removed + day_removed))
  grand_net=$((grand_net + net_change))
  grand_change=$((grand_change + change_total))

  printf "%-12s %10d %10d %10d %12d %14d %8d\n" \
    "$day" "$day_added" "$day_modified" "$day_removed" "$net_change" "$change_total" "$changed_repos"
done

printf "%-12s %10d %10d %10d %12d %14d %8s\n" \
  "TOTAL" "$grand_added" "$grand_modified" "$grand_removed" "$grand_net" "$grand_change" "-"
