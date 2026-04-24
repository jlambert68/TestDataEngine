import type { DataSourceListItem, FilterExpression, FilterRequest } from '../types/api'

export function makeFilterRequest(ds: DataSourceListItem, expr: FilterExpression): FilterRequest {
  return {
    SchemaVersion: '1.0',
    RequestUuid: crypto.randomUUID(),
    DataSourceUuid: ds.dataSourceUuid,
    DataSourceName: ds.dataSourceName,
    RequestFilter: expr,
  }
}
