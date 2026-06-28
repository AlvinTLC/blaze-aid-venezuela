# BlazeAid Hub — API Readiness Audit

> Estado al cierre de Fase 3 (commit `e6298e7`). Propósito: dejar el API lo más
> listo posible para ayuda humanitaria y definir el contrato que el frontend
> consumirá, **antes** de levantar un agente de FE.

## ✅ Lo que ya está (Fases 0–3)

- Stack: Go 1.25 + Huma v2 (chi) + pgx/Postgres (TimescaleDB + pgvector) + Redis
  + River + Docker distroless. `docker compose up` levanta `api` + `worker` + db.
- **Escritura**: `POST /api/v1/ingest/{project,resource,missing,volunteer}` —
  upsert idempotente por `(source, external_id)`, protegido con Bearer JWT.
- **Sync (delta)**: `GET /api/v1/sync?since=&limit=` — cursor por `updated_at`.
- **Webhooks async**: `POST /api/v1/webhook/{source}` → `webhooks_log` + River job
  (exactly-once) → worker rutea a tabla tipada + `events` hypertable.
- **Auth**: `POST /api/v1/magic-login` → `POST /api/v1/auth/verify` → session JWT.
- Tests de integración (testcontainers) verdes; hardening de prod (no secretos
  default), OpenAPI/Swagger en `/docs`.

## 🔴 P0 — Bloqueos duros para que el FE pueda construir

### 1. NO existen endpoints de lectura (browse / search / detail)
Hoy el único camino de lectura es `/sync` (un delta para replicación, no una API
de consulta). El FE de una plataforma humanitaria necesita **listar, filtrar,
buscar y ver detalle**. Sin esto, no hay nada que mostrar.

**Contrato propuesto (lo que el FE va a consumir):**

```
GET /api/v1/projects?region=&status=&category=&q=&limit=&offset=
GET /api/v1/projects/{id}
GET /api/v1/resources?region=&type=&status=&q=&limit=&offset=
GET /api/v1/resources/{id}
GET /api/v1/missing?region=&status=&q=&limit=&offset=
GET /api/v1/missing/{id}
GET /api/v1/volunteers?region=&skill=&status=&q=&limit=&offset=
GET /api/v1/volunteers/{id}
```

Respuesta de lista (forma estable para el FE):
```jsonc
{
  "items": [ /* entidad tipada */ ],
  "total": 128,          // total que matchea el filtro (para paginación)
  "limit": 20,
  "offset": 0
}
```
- Filtros: `region`, `status`, `category`/`type`/`skill`, `q` (texto en
  title/name/full_name/description). `limit` default 20, máx 100. `offset` para paginar.
- Lecturas **públicas** (el catálogo de ayuda es público). Excepción: ver más abajo PII.

### 2. CORS
No hay middleware CORS. Un navegador en otro origen (el FE) no puede llamar al
API. Hace falta `go-chi/cors` con orígenes permitidos por env
(`CORS_ORIGINS`), métodos GET/POST/OPTIONS y header `Authorization`.

## 🟠 P1 — Production-readiness (para uso humanitario real)

- **Migraciones en deploy**: `migrations/001_init.sql` solo corre vía `initdb.d`
  en el **primer** boot de un volumen vacío. En un Postgres gestionado (deploy)
  **no hay migrador**. Añadir un runner en boot (golang-migrate/goose, o aplicar
  los `.sql` embebidos con `embed.FS`) para que el schema exista y evolucione en prod.
- **Rate limiting + tamaño de body**: API público → añadir `httprate` por IP y
  `http.MaxBytesReader` (p.ej. 1 MB) para evitar abuso/DoS.
- **PII / privacidad (crítico en humanitario)**: `missing_persons` y `volunteers`
  contienen datos personales (nombre, contacto, foto). Decidir: ¿`contact` y
  `photo_url` se exponen en lecturas públicas o requieren auth? Recomendado:
  ocultar `contact` en listas públicas, exponerlo solo a usuarios autenticados.
- **Observabilidad**: logging estructurado de request (status, latencia, ruta) y
  endpoint de métricas (`/metrics` Prometheus) para operar bajo carga.
- **Deploy + CI/CD**: target de hosting (Fly/Railway/VPS) + GitHub Actions
  (test con testcontainers, build imagen distroless, push). Healthz ya existe.
- **Validación de dominio**: rangos lat/lng, enums de `status`, normalización de
  `region` (catálogo de estados de Venezuela) para que los filtros del FE sean fiables.

## 🟢 P2 — Mejoras posteriores

- Búsqueda semántica con embeddings (columna pgvector ya provista).
- Analytics/dashboard: counts por región/estado, recientes, series temporales
  sobre el hypertable `events`.
- `GET /api/v1/admin/jobs` (monitoreo River) + firma HMAC por proveedor en webhooks.
- Realtime (WS/SSE) para el feed en vivo de personas desaparecidas / recursos.

## 🎯 Recomendación de Fase 4 (orden)

1. **Read API + paginación + filtros** (P0.1) — desbloquea al FE.
2. **CORS** (P0.2) — desbloquea al navegador.
3. **Migraciones en boot** (P1) — necesario para desplegar.
4. **PII gating + rate limit + body cap** (P1) — seguro para público.
5. **Deploy + CI/CD** — link vivo para el Google Form.

Con (1)+(2) listos, **un agente de FE ya puede arrancar contract-first** contra
el OpenAPI en `/openapi.json`, en paralelo a (3)–(5).
