# Reglas del Proyecto: sync-product-image

## Información General
- **Lenguaje**: Go 1.23+
- **Tipo**: Herramienta CLI batch para sincronización de imágenes de productos
- **Backend**: Supabase (PostgREST API + Storage Buckets + PostgreSQL directo vía GORM)
- **Dominio**: E-commerce — sistema de gestión de productos para Balys Tiendas

## Estructura del Proyecto

```
sync-product-image/
├── main.go              # Punto de entrada, conversión de imágenes, upload y flujo principal
├── database/sb.go       # Capa de datos: struct Supabase, conexiones, queries
├── model/Product.go     # Modelo GORM/JSON del producto
├── model/Media.go       # Modelo GORM/JSON de media (imágenes)
├── assets/original/     # Imágenes fuente (PNG/JPG) - gitignored
├── assets/webp/         # Caché de imágenes convertidas a WebP - gitignored
├── .env                 # Variables de entorno
└── .env.example         # Plantilla de configuración
```

## Reglas de Código

### 1. Convenciones Go
- Usar `os.WriteFile` en lugar de `ioutil.WriteFile` (deprecado desde Go 1.16)
- Evitar `io/ioutil` — usar `io` y `os` directamente
- Las funciones deben retornar `error` y no usar `panic()` excepto en fallos irrecuperables de inicialización
- Usar `context.Context` como primer parámetro en funciones con I/O externo

### 2. Variables de Entorno
- **NUNCA** hardcodear credenciales, DSNs, o claves API en el código fuente
- Todas las configuraciones sensibles deben venir de `.env` vía `godotenv`
- Las variables de entorno requeridas son:
  - `SUPABASE_URL`, `SUPABASE_KEY` — conexión al cliente Supabase
  - `SUPABASE_PUBLIC_MEDIA_URL` — URL pública del bucket de media
  - `API_BASE_URL`, `ENDPOINT_MEDIA` — API endpoints
  - `WATCH_FILE_PATH` — directorio de imágenes fuente
  - `OUTPUT_FILE_PATH` — directorio de caché WebP
  - `sp_host`, `sp_user`, `sp_password`, `sp_port`, `sp_dbname` — conexión PostgreSQL directa

### 3. Seguridad
- `.env` **DEBE** estar en `.gitignore` — nunca versionar credenciales reales
- Binarios compilados (`sync-product-image`, `.exe`) **NO** deben versionarse
- Archivos CSV con datos de producción **NO** deben versionarse
- Las claves Supabase (anon key, service role key) nunca deben aparecer en el código fuente

### 4. Modelos de Datos
- Los modelos usan dual tagging: `gorm:"..."` + `json:"..."`
- Los campos nullable usan punteros (`*uint`, `*string`)
- Cada modelo debe implementar `TableName() string` para GORM
- El campo `product_id` es la PK de Product; `media_id` es la PK de Media
- `image_id` en Product es FK a `media_id` en Media

### 5. Capa de Base de Datos
- El proyecto usa **dos métodos de acceso** a la DB:
  - **Supabase Client (PostgREST)**: para queries principales (`GetProducts`, `GetMedia`, inserts, updates)
  - **GORM (conexión directa)**: disponible pero actualmente secundario
- Los queries paginados usan `.Range(offset, offset+999)` con bloques de 1000
- La struct `Supabase` en `database/sb.go` encapsula ambos clientes

### 6. Procesamiento de Imágenes
- Formatos soportados de entrada: PNG, JPG
- Formato de salida: WebP (calidad 80%)
- Se mantiene un caché local en `OUTPUT_FILE_PATH` para evitar re-conversiones
- Las imágenes se suben al bucket `media` de Supabase en la ruta `/products/assets/webp/{sku}.webp`
- El MIME type de upload siempre es `image/webp`

### 7. Timestamps
- Usar `time.Now().UTC().Format(time.RFC3339)` o un formato explícito
- **NO** usar `strings.Split(time.Now().UTC().String(), " +")[0]` — es frágil
- Los campos de auditoría son: `created_at`, `created_by`, `updated_at`, `updated_by`

### 8. Manejo de Errores
- Los errores de inicialización (env, cliente Supabase) usan `log.Fatal`
- Los errores por producto individual deben loggearse y continuar con `continue` (no detener el batch)
- Implementar logging estructurado con niveles (info, warn, error) en futuras mejoras

### 9. Git / Versionamiento
- `.gitignore` debe incluir: `/assets`, `/sincronizador`, `.env`, `*.exe`, `sync-product-image`, `*.csv`
- No versionar binarios compilados ni datos de producción
- Usar `.env.example` como plantilla para documentar variables requeridas

### 10. Dependencias
- Eliminar dependencias no utilizadas (`go.mongodb.org/mongo-driver`)
- Ejecutar `go mod tidy` después de cambios en imports
- Mantener dependencias actualizadas con `go get -u`
