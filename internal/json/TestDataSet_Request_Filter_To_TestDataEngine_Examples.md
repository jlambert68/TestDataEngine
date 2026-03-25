# RequestFilter Request Schema Examples

Below is a set of **valid example payloads** that covers the supported shapes in the schema:

- top-level request
- every operator
- scalar and array values
- `and`, `or`, `not`
- nested expressions
- special constraints for `exists`, `in`/`nin`, and string operators

## 1. Minimal valid request with `eq`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "country",
    "op": "eq",
    "value": "SE"
  }
}
```

## 2. With optional `schemaVersion`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "field": "status",
    "op": "eq",
    "value": "Open"
  }
}
```

---

# Comparison operator examples

## 3. `eq`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Products",
  "RequestFilter": {
    "field": "category",
    "op": "eq",
    "value": "Books"
  }
}
```

## 4. `neq`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Products",
  "RequestFilter": {
    "field": "category",
    "op": "neq",
    "value": "Electronics"
  }
}
```

## 5. `gt`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "field": "totalAmount",
    "op": "gt",
    "value": 100
  }
}
```

## 6. `gte`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "field": "totalAmount",
    "op": "gte",
    "value": 100
  }
}
```

## 7. `lt`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Inventory",
  "RequestFilter": {
    "field": "stock",
    "op": "lt",
    "value": 10
  }
}
```

## 8. `lte`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Inventory",
  "RequestFilter": {
    "field": "stock",
    "op": "lte",
    "value": 10
  }
}
```

## 9. `in`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "country",
    "op": "in",
    "value": ["SE", "NO", "DK"]
  }
}
```

## 10. `nin`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "country",
    "op": "nin",
    "value": ["RU", "BY"]
  }
}
```

## 11. `contains`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "email",
    "op": "contains",
    "value": "@example.com"
  }
}
```

## 12. `startsWith`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "lastName",
    "op": "startsWith",
    "value": "Lam"
  }
}
```

## 13. `endsWith`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Files",
  "RequestFilter": {
    "field": "fileName",
    "op": "endsWith",
    "value": ".pdf"
  }
}
```

## 14. `exists` = true

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "phoneNumber",
    "op": "exists",
    "value": true
  }
}
```

## 15. `exists` = false

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "middleName",
    "op": "exists",
    "value": false
  }
}
```

---

# Scalar value type examples

## 16. String value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Employees",
  "RequestFilter": {
    "field": "department",
    "op": "eq",
    "value": "Engineering"
  }
}
```

## 17. Number value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Sensors",
  "RequestFilter": {
    "field": "temperature",
    "op": "gt",
    "value": 21.5
  }
}
```

## 18. Integer value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Employees",
  "RequestFilter": {
    "field": "age",
    "op": "gte",
    "value": 18
  }
}
```

## 19. Boolean value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Users",
  "RequestFilter": {
    "field": "isActive",
    "op": "eq",
    "value": true
  }
}
```

## 20. Null value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "field": "deletedAt",
    "op": "eq",
    "value": null
  }
}
```

## 21. Array of scalar values

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Tickets",
  "RequestFilter": {
    "field": "priority",
    "op": "in",
    "value": ["High", "Medium", "Low"]
  }
}
```

---

# Logical expression examples

## 22. `and`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "and": [
      {
        "field": "status",
        "op": "eq",
        "value": "Paid"
      },
      {
        "field": "totalAmount",
        "op": "gte",
        "value": 500
      }
    ]
  }
}
```

## 23. `or`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "or": [
      {
        "field": "country",
        "op": "eq",
        "value": "SE"
      },
      {
        "field": "country",
        "op": "eq",
        "value": "NO"
      }
    ]
  }
}
```

## 24. `not`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Users",
  "RequestFilter": {
    "not": {
      "field": "isBlocked",
      "op": "eq",
      "value": true
    }
  }
}
```

---

# Nested expression examples

## 25. `and` containing `or`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "and": [
      {
        "or": [
          {
            "field": "status",
            "op": "eq",
            "value": "Open"
          },
          {
            "field": "status",
            "op": "eq",
            "value": "Pending"
          }
        ]
      },
      {
        "field": "totalAmount",
        "op": "gt",
        "value": 50
      }
    ]
  }
}
```

## 26. `or` containing `and`

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "or": [
      {
        "and": [
          {
            "field": "country",
            "op": "eq",
            "value": "SE"
          },
          {
            "field": "isActive",
            "op": "eq",
            "value": true
          }
        ]
      },
      {
        "and": [
          {
            "field": "country",
            "op": "eq",
            "value": "NO"
          },
          {
            "field": "isActive",
            "op": "eq",
            "value": true
          }
        ]
      }
    ]
  }
}
```

## 27. `not` applied to a group

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Products",
  "RequestFilter": {
    "not": {
      "or": [
        {
          "field": "category",
          "op": "eq",
          "value": "Discontinued"
        },
        {
          "field": "stock",
          "op": "lte",
          "value": 0
        }
      ]
    }
  }
}
```

## 28. Deeply nested mixed example

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Customers",
  "RequestFilter": {
    "and": [
      {
        "field": "isActive",
        "op": "eq",
        "value": true
      },
      {
        "or": [
          {
            "field": "country",
            "op": "in",
            "value": ["SE", "NO", "DK"]
          },
          {
            "field": "email",
            "op": "endsWith",
            "value": ".org"
          }
        ]
      },
      {
        "not": {
          "field": "lastName",
          "op": "startsWith",
          "value": "Test"
        }
      },
      {
        "field": "phoneNumber",
        "op": "exists",
        "value": true
      }
    ]
  }
}
```

---

# Operator-focused examples for special rules

## 29. `contains` must use string value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Articles",
  "RequestFilter": {
    "field": "title",
    "op": "contains",
    "value": "schema"
  }
}
```

## 30. `startsWith` must use string value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Articles",
  "RequestFilter": {
    "field": "slug",
    "op": "startsWith",
    "value": "api-"
  }
}
```

## 31. `endsWith` must use string value

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Articles",
  "RequestFilter": {
    "field": "slug",
    "op": "endsWith",
    "value": "-guide"
  }
}
```

## 32. `in` with mixed scalar array types

This is allowed by the schema because array items are `scalarValue`.

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "GenericData",
  "RequestFilter": {
    "field": "mixedField",
    "op": "in",
    "value": ["A", 1, true, null]
  }
}
```

## 33. `nin` with numeric array

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "field": "statusCode",
    "op": "nin",
    "value": [400, 404, 500]
  }
}
```

---

# One “kitchen sink” example

This one exercises most of the schema in a single payload.

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceUuid": "a79c6e3b-f726-419d-b0d0-014203798bb2",
  "DataSourceName": "Orders",
  "RequestFilter": {
    "and": [
      {
        "field": "status",
        "op": "in",
        "value": ["Paid", "Shipped", "Delivered"]
      },
      {
        "field": "totalAmount",
        "op": "gte",
        "value": 100
      },
      {
        "field": "customerEmail",
        "op": "contains",
        "value": "@company.com"
      },
      {
        "or": [
          {
            "field": "country",
            "op": "eq",
            "value": "SE"
          },
          {
            "field": "country",
            "op": "eq",
            "value": "NO"
          }
        ]
      },
      {
        "not": {
          "field": "notes",
          "op": "startsWith",
          "value": "TEST"
        }
      },
      {
        "field": "trackingNumber",
        "op": "exists",
        "value": true
      }
    ]
  }
}
```

# Notes

- `schemaVersion` is optional.
- `DataSource` and `RequestFilter` are required.
- top-level extra properties are not allowed.
- comparison objects require `field` and `op`.
- `value` is required for all operators, including `exists`.
- `exists` requires `value` to be a boolean.
- `in` and `nin` require `value` to be a non-empty array.
- `contains`, `startsWith`, and `endsWith` require `value` to be a string.
- other operators can use any `scalarValue`: string, number, integer, boolean, or null.
- arrays may contain mixed scalar types because `scalarValue` allows that.
- in JSON Schema, `integer` is effectively a subtype of `number`, so having both in the union is redundant but valid.

