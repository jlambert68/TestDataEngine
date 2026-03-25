package filtersql

import (
	"encoding/json"
	"fmt"
)

func ExampleCompiler_Compile() {
	input := []byte(`{
	  "SchemaVersion": "1.0",
	  "RequestUuid": "11111111-1111-4111-8111-111111111111",
	  "DataSourceUuid": "22222222-2222-4222-8222-222222222222",
	  "DataSourceName": "people",
	  "RequestFilter": {
	    "and": [
	      { "field": "age", "op": "gte", "value": 18 },
	      {
	        "or": [
	          { "field": "city", "op": "eq", "value": "Stockholm" },
	          { "field": "city", "op": "eq", "value": "Malmo" }
	        ]
	      },
	      { "field": "name", "op": "startsWith", "value": "Jo" }
	    ]
	  }
	}`)

	var req Request
	if err := json.Unmarshal(input, &req); err != nil {
		panic(err)
	}

	compiler := Compiler{
		Placeholder: Dollar,
		QuoteIdent:  true,
	}

	where, args, err := compiler.Compile(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(where)
	fmt.Println(args)

	// Output:
	// ("age" >= $1) AND (("city" = $2) OR ("city" = $3)) AND ("name" LIKE $4)
	// [18 Stockholm Malmo Jo%]
}
