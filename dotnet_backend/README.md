# dotnet_backend

ASP.NET Core backend that mirrors the Go web API used by the existing Vue UI.

## Endpoints

- `GET /api/v1/datasources`
- `GET /api/v1/datasources/{id}/fields?source=...`
- `GET /api/v1/datasources/{id}/facets?source=...&field=...&limit=...&q=...`
- `POST /api/v1/query/preview`
- `GET /api/v1/healthz`

Non-API routes serve static files from `ui/dist` with SPA fallback to `index.html`.

## Run

Requires `.NET 8 SDK`.

```bash
cd dotnet_backend
dotnet restore
dotnet run
```

Default bind address is `http://0.0.0.0:8080`.

Override with:

```bash
HTTP_ADDR=:8081 dotnet run
```

## Notes

- The API contract matches `ui/src/types/api.ts`, so the UI can switch from Go backend to this backend without UI code changes.
- Runtime filter request fields keep Pascal-case JSON (`SchemaVersion`, `RequestUuid`, etc.).
- Web wrapper fields use lower camel case (`source`, `compiledWhereSql`, `dataSet`, etc.).
