# TestDataEngine .NET Rewrite Requirements

This folder defines the minimum contract another developer must implement to recreate the current Go project's behavior in C#/.NET, including unit-test expectations.

Use these files in this order:

1. [01-runtime-requirements.md](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/01-runtime-requirements.md)
2. [02-contract-details.md](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/02-contract-details.md)
3. [03-unit-test-requirements.md](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/03-unit-test-requirements.md)
4. [models/TestDataEngineContracts.cs](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/models/TestDataEngineContracts.cs)
5. [models/TestDataEnginePorts.cs](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/models/TestDataEnginePorts.cs)
6. [sql/sqlite-schema.sql](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/sql/sqlite-schema.sql)
7. [sql/postgres-schema.sql](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/sql/postgres-schema.sql)
8. [examples/filter-request.sample.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/examples/filter-request.sample.json)
9. [examples/dataset-response.sample.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/examples/dataset-response.sample.json)
10. [TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/internal/json/TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json)
11. [TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/internal/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json)
12. [TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine_Examples.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/internal/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine_Examples.json)

Schema authority:

- Only the root-level files directly under [internal/json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/internal/json) are authoritative.
- Do not use `internal/json/old`, `P26_2`, `testdata/pi26_2`, or `dot-net-requirements/json` as the source of truth.
- Files under [dot-net-requirements/json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/json) are documentation snapshots only.

Reference snapshots included in this pack:

- [TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/json/TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json)
- [TestDataSet_Request_Filter_To_TestDataEngine_Examples.md](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/json/TestDataSet_Request_Filter_To_TestDataEngine_Examples.md)
- [TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json)
- [TestDataSet_Response_From_TestDataEngine_Examples.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/json/TestDataSet_Response_From_TestDataEngine_Examples.json)
- [example-output.json](/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/dot-net-requirements/json/example-output.json)

Authoritative behavior for the rewrite:

- Runtime engine behavior in `internal/filters`, `internal/sqlitecsv`, `internal/logging`, and `cmd/testdataengine`
- Typed compiler behavior in `internal/filtersql`
- Existing unit tests

Important compatibility note:

- The Go repository has two filter-contract layers: `internal/filters` and `internal/filtersql`.
- They are close, but not identical.
- The .NET rewrite should preserve both layers and their current differences because the unit tests depend on them.
