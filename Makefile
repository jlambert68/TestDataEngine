GO ?= go
PKG ?= ./...
CMD ?= ./cmd/testdataengine

CSV_PATH ?= testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv
SQLITE_DB ?= testdata/SQLiteDB/identifier.sqlite
SQLITE_TABLE ?= main.data_items
MAX_ITEMS ?= 2
RANDOM_SEED_GUID ?=

# Preset parameter values.
CSV_PATH_FULL ?= testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv
CSV_PATH_SMALL ?= testdata/pi26_2/FenixRawTestdata_3rows_240705.csv
SQLITE_DB_DEFAULT ?= testdata/SQLiteDB/identifier.sqlite
SQLITE_TABLE_DEFAULT ?= main.data_items
MAX_ITEMS_DEFAULT ?= 2
MAX_ITEMS_SMALL ?= 1
SEED_DETERMINISTIC_A ?= bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
SEED_DETERMINISTIC_B ?= aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa

# Matched CSV/DB inputs for comparable output.
MATCH_CSV_PATH ?= $(CSV_PATH_FULL)
MATCH_SQLITE_DB ?= $(SQLITE_DB_DEFAULT)
MATCH_SQLITE_TABLE ?= $(SQLITE_TABLE_DEFAULT)
MATCH_MAX_ITEMS ?= 2
MATCH_RANDOM_SEED_GUID ?= $(SEED_DETERMINISTIC_A)

.PHONY: test test-verbose testv run-csv run-db run-csv-default run-csv-small run-db-default run-db-seeded run-csv-match run-db-match

test:
	$(GO) test $(PKG)

test-verbose:
	$(GO) test -v $(PKG)

testv:test-verbose

# Start TestDataEngine using CSV as source.
run-csv:
	$(GO) run $(CMD) -source csv -csv "$(CSV_PATH)" -max-items $(MAX_ITEMS) -random-seed-guid "$(RANDOM_SEED_GUID)"

# Start TestDataEngine using SQLite database as source.
run-db:
	$(GO) run $(CMD) -source sqlite -sqlite-db "$(SQLITE_DB)" -sqlite-table "$(SQLITE_TABLE)" -max-items $(MAX_ITEMS) -random-seed-guid "$(RANDOM_SEED_GUID)"

# CSV preset: full dataset with deterministic seed.
run-csv-default: CSV_PATH := $(CSV_PATH_FULL)
run-csv-default: MAX_ITEMS := $(MAX_ITEMS_DEFAULT)
run-csv-default: RANDOM_SEED_GUID := $(SEED_DETERMINISTIC_A)
run-csv-default: run-csv

# CSV preset: small dataset with deterministic seed.
run-csv-small: CSV_PATH := $(CSV_PATH_SMALL)
run-csv-small: MAX_ITEMS := $(MAX_ITEMS_SMALL)
run-csv-small: RANDOM_SEED_GUID := $(SEED_DETERMINISTIC_B)
run-csv-small: run-csv

# DB preset: default table without deterministic seed.
run-db-default: SQLITE_DB := $(SQLITE_DB_DEFAULT)
run-db-default: SQLITE_TABLE := $(SQLITE_TABLE_DEFAULT)
run-db-default: MAX_ITEMS := $(MAX_ITEMS_DEFAULT)
run-db-default: RANDOM_SEED_GUID :=
run-db-default: run-db

# DB preset: default table with deterministic seed.
run-db-seeded: SQLITE_DB := $(SQLITE_DB_DEFAULT)
run-db-seeded: SQLITE_TABLE := $(SQLITE_TABLE_DEFAULT)
run-db-seeded: MAX_ITEMS := $(MAX_ITEMS_DEFAULT)
run-db-seeded: RANDOM_SEED_GUID := $(SEED_DETERMINISTIC_A)
run-db-seeded: run-db

# CSV preset: matched settings for CSV vs DB output comparisons.
run-csv-match: CSV_PATH := $(MATCH_CSV_PATH)
run-csv-match: MAX_ITEMS := $(MATCH_MAX_ITEMS)
run-csv-match: RANDOM_SEED_GUID := $(MATCH_RANDOM_SEED_GUID)
run-csv-match: run-csv

# DB preset: matched settings for CSV vs DB output comparisons.
run-db-match: SQLITE_DB := $(MATCH_SQLITE_DB)
run-db-match: SQLITE_TABLE := $(MATCH_SQLITE_TABLE)
run-db-match: MAX_ITEMS := $(MATCH_MAX_ITEMS)
run-db-match: RANDOM_SEED_GUID := $(MATCH_RANDOM_SEED_GUID)
run-db-match: run-db
