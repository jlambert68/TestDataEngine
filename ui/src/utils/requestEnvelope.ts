import type { DataSourceListItem, FilterExpression, FilterRequest } from '../types/api'

export function makeFilterRequest(ds: DataSourceListItem, expr: FilterExpression, requestUuid?: string): FilterRequest {
  return {
    SchemaVersion: '1.0',
    RequestUuid: requestUuid || crypto.randomUUID(),
    DataSourceUuid: ds.dataSourceUuid,
    DataSourceName: ds.dataSourceName,
    RequestFilter: expr,
  }
}
