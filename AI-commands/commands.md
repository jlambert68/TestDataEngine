# AI Commands - MQ Gateway

This file contains command sequences to keep **unit tests**, **diagrams**, and **documentation** aligned.

Run commands from:

```bash
cd mq-gateway
```

## 0. Update Missing Docs/Diagrams (Run From Repo Root)

Run from repository root (`MQDockerContainer`). This creates missing docs placeholders only (does not overwrite existing files):

```bash
set -e

# Create missing docs/<name>.md and docs/<name>.activity.puml for every .go file in mq-gateway
for f in $(find mq-gateway -type f -name '*.go' | sort); do
  d="$(dirname "$f")/docs"
  b="$(basename "$f" .go)"
  m="$d/$b.md"
  p="$d/$b.activity.puml"

  mkdir -p "$d"

  if [ ! -f "$m" ]; then
    cat > "$m" <<EOF
# $b.go

Source: \`$f\`

## Purpose

TODO: describe behavior and responsibilities.
EOF
  fi

  if [ ! -f "$p" ]; then
    cat > "$p" <<'EOF'
@startuml
start
:TODO add activity flow;
stop
@enduml
EOF
  fi
done

# Create missing docs/<test_name>.md for each *_test.go
for f in $(find mq-gateway -type f -name '*_test.go' | sort); do
  d="$(dirname "$f")/docs"
  b="$(basename "$f" .go)"
  m="$d/$b.md"

  mkdir -p "$d"

  if [ ! -f "$m" ]; then
    cat > "$m" <<EOF
# $b.go

Source: \`$f\`

## Test coverage

TODO: describe key test cases and assertions.
EOF
  fi
done
```

## 1. Run Unit Tests

### Local (host)

```bash
make Run_UnitTests_Local
```

### Containerized (portable)

```bash
make Run_UnitTests_docker-compose.debug.mq.local
```

### Containerized verbose

```bash
make Run_UnitTests_Verbose_docker-compose.debug.mq.local
```

## 2. Verify Diagram/Doc Presence Per Go File

Each `.go` file should have:

- `docs/<name>.activity.puml`
- `docs/<name>.md`

```bash
for f in $(find . -type f -name '*.go' | sort); do
  d=$(dirname "$f")/docs
  b=$(basename "$f" .go)
  p="$d/$b.activity.puml"
  m="$d/$b.md"
  [ -f "$p" ] || echo "MISSING $p"
  [ -f "$m" ] || echo "MISSING $m"
done
```

## 3. Verify Test-Doc Presence Per Test File

Each `*_test.go` file should have:

- `docs/<test_name>.md`

```bash
for f in $(find . -type f -name '*_test.go' | sort); do
  d=$(dirname "$f")/docs
  b=$(basename "$f" .go)
  m="$d/$b.md"
  [ -f "$m" ] || echo "MISSING $m"
done
```

## 4. Verify Main User/Diagram Docs Exist

```bash
test -f ../docs/mq-gateway-user-guide.md || echo "MISSING ../docs/mq-gateway-user-guide.md"
test -f ../docs/diagrams/mqrest-client-sequence.puml || echo "MISSING ../docs/diagrams/mqrest-client-sequence.puml"
```

## 4.1 Verify Root docs/ Folder Quality (Run From Repo Root)

Run from repository root (`MQDockerContainer`) to validate only `/docs`:

```bash
set -e

echo "== Root docs files =="
find docs -maxdepth 3 -type f | sort

echo "== Root docs counts =="
echo "all: $(find docs -type f | wc -l)"
echo "md: $(find docs -type f -name '*.md' | wc -l)"
echo "puml: $(find docs -type f -name '*.puml' | wc -l)"

echo "== Required root docs =="
test -f docs/mq-gateway-user-guide.md || echo "MISSING docs/mq-gateway-user-guide.md"
test -f docs/diagrams/mqrest-client-sequence.puml || echo "MISSING docs/diagrams/mqrest-client-sequence.puml"

echo "== Empty files =="
find docs -type f -size 0 -print

echo "== Placeholder markers (TODO/TBD/FIXME) =="
grep -RIn 'TODO\|TBD\|FIXME' docs || true

echo "== Markdown title check (# ...) =="
for f in $(find docs -type f -name '*.md' | sort); do
  head -n 1 "$f" | grep -q '^# ' || echo "BAD_MD_TITLE $f"
done

echo "== PlantUML boundary checks =="
for f in $(find docs -type f -name '*.puml' | sort); do
  grep -q '^@startuml' "$f" || echo "BAD_PUML_START $f"
  grep -q '^@enduml' "$f" || echo "BAD_PUML_END $f"
done
```

## 5. Quick Drift Check (What Changed)

```bash
git status --short
```

## 5.1 Change-Driven Sync Command (Do This First)

Use this command block to detect what changed, then update dependent artifacts:

```bash
set -e

echo "== Changed files =="
git status --short

echo "== Source change detection =="
code_changed=$(git status --porcelain | awk '{print $2}' | grep -E '\.go$' || true)
docs_changed=$(git status --porcelain | awk '{print $2}' | grep -E '\.md$' || true)
diag_changed=$(git status --porcelain | awk '{print $2}' | grep -E '\.puml$' || true)

if [ -n "$code_changed" ]; then
  echo "CODE changed -> update docs/*.md, docs/*.activity.puml, and *_test.go as needed."
fi
if [ -n "$docs_changed" ]; then
  echo "DOCS changed -> verify/update related .go behavior docs and matching diagrams/tests."
fi
if [ -n "$diag_changed" ]; then
  echo "DIAGRAMS changed -> verify/update corresponding .md docs and source implementation notes."
fi
```

## 6. Full Consistency Check (Recommended)

```bash
set -e

make Run_UnitTests_Local

for f in $(find . -type f -name '*.go' | sort); do
  d=$(dirname "$f")/docs
  b=$(basename "$f" .go)
  test -f "$d/$b.activity.puml"
  test -f "$d/$b.md"
done

for f in $(find . -type f -name '*_test.go' | sort); do
  d=$(dirname "$f")/docs
  b=$(basename "$f" .go)
  test -f "$d/$b.md"
done

test -f ../docs/mq-gateway-user-guide.md

echo "OK: tests, diagrams, and documentation are in sync."
```

## 6.1 Update-Order Rule (Required)

When any file changes, follow this order:

1. `git status --short` to identify source of change (`.go`, `.md`, `.puml`).
2. Update dependent artifacts:
   - If source is `.go`: update matching docs + diagrams + tests.
   - If source is `.md`: update related diagrams/code notes/tests where needed.
   - If source is `.puml`: update related docs and confirm code behavior still matches.
3. Update unit tests last.
4. Run unit tests:

```bash
make Run_UnitTests_Local
```

## 7. AI Command Prompt Template

Use this prompt when updating code:

```text
Update mq-gateway code and keep tests, activity diagrams (.activity.puml), and per-file docs (.md) in sync.
After edits:
1) run make Run_UnitTests_Local
2) ensure every .go has docs/<name>.activity.puml and docs/<name>.md
3) ensure every *_test.go has docs/<test_name>.md
4) show git status --short and summarize changed files.
```
