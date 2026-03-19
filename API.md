# API Red - Documentación para cliente Android

Base URL del servidor: `http://35.199.180.75:8080`

---

## GET /bus-stop/:stopid

Retorna información en tiempo real de un paradero: nombre, servicios que pasan por él y tiempo de llegada de los próximos buses.

Los códigos de paradero tienen el formato `PA433`, `PD1418`, etc.

### Respuesta

```json
{
  "id": "PA433",
  "name": "PARADA 1 / ESCUELA DE INGENIERÍA",
  "services": [
    {
      "id": "506e",
      "valid": true,
      "status_description": "Información de tiempos de los próximos 2 buses",
      "buses": [
        {
          "id": "VJBC-90",
          "meters_distance": 4147,
          "min_arrival_time": 15,
          "max_arrival_time": 19
        },
        {
          "id": "VJJV-38",
          "meters_distance": 4659,
          "min_arrival_time": 17,
          "max_arrival_time": 21
        }
      ]
    },
    {
      "id": "506",
      "valid": true,
      "status_description": "Parada inhabilitada, favor dirigirse al Paradero PA434",
      "buses": []
    }
  ]
}
```

### Campos

| Campo | Tipo | Descripción |
|---|---|---|
| `id` | string | Código del paradero |
| `name` | string | Nombre del paradero |
| `services` | array | Servicios que operan en este paradero |
| `services[].id` | string | Número de servicio (ej: "D01", "412") |
| `services[].valid` | bool | Siempre `true` — todos los servicios retornados operan en el paradero |
| `services[].status_description` | string | Descripción del estado actual del servicio |
| `services[].buses` | array | Próximos buses en camino (0, 1 o 2 elementos) |
| `buses[].id` | string | Patente del bus |
| `buses[].meters_distance` | int | Distancia en metros al paradero |
| `buses[].min_arrival_time` | int | Tiempo mínimo de llegada en minutos |
| `buses[].max_arrival_time` | int | Tiempo máximo de llegada en minutos |

### Casos especiales

- `buses` vacío → no hay buses en camino para ese servicio actualmente
- `max_arrival_time: 99` → el bus está a más de cierta cantidad de minutos (el predictor indica "más de X min"); mostrar como "> X min"
- `status_description` puede contener mensajes como "Parada inhabilitada" o "No hay buses que se dirijan al paradero" — útil para mostrarle al usuario

---

## GET /metro-network

Retorna el estado actual de toda la red de metro: líneas, estaciones, estado operativo y horarios.

### Respuesta

```json
{
  "api_status": "OK",
  "time": "2026-03-17 21:49:38",
  "issues": false,
  "lines": [
    {
      "name": "Línea 1",
      "id": "L1",
      "issues": false,
      "stations_closed_by_schedule": 0,
      "stations": [
        {
          "name": "San Pablo",
          "id": "san-pablo",
          "status": 0,
          "lines": ["L1", "L5"],
          "description": "Estación Operativa",
          "reason": "",
          "is_closed_by_schedule": false,
          "schedule": {
            "open": {
              "weekdays": "06:00",
              "saturday": "06:30",
              "holidays": "08:00"
            },
            "close": {
              "weekdays": "23:00",
              "saturday": "23:00",
              "holidays": "23:00"
            }
          }
        }
      ]
    }
  ]
}
```

### Campos

| Campo | Tipo | Descripción |
|---|---|---|
| `api_status` | string | `"OK"` o mensaje de error |
| `time` | string | Timestamp de la consulta |
| `issues` | bool | `true` si alguna línea o estación tiene problemas |
| `lines[].id` | string | ID de línea: `"L1"`, `"L2"`, `"L3"`, `"L4"`, `"L4A"`, `"L5"`, `"L6"` |
| `lines[].issues` | bool | `true` si la línea tiene estaciones con problemas |
| `lines[].stations_closed_by_schedule` | int | Cantidad de estaciones cerradas por horario |
| `stations[].status` | int | `0` = operativa, `1` = problema parcial, `2` = cerrada, `3` = desconocido |
| `stations[].lines` | array | Líneas que pasan por la estación (útil para transbordos) |
| `stations[].is_closed_by_schedule` | bool | `true` si la estación está fuera de su horario de operación |
| `stations[].schedule` | object | Horarios de apertura y cierre por tipo de día |
| `stations[].reason` | string | Motivo del problema, si aplica |

---

## Notas generales

- El servidor no tiene autenticación — acceso directo por IP y puerto
- El endpoint `/bus-stop` hace una request en tiempo real a red.cl en cada llamada — no cachea resultados
- El endpoint `/metro-network` sí cachea: los horarios de estaciones se actualizan a la 1am y el estado feriado a medianoche; el estado operativo de las estaciones se consulta en tiempo real a metro.cl
- Si `api_status` es distinto de `"OK"` en `/metro-network`, hubo un error al conectarse a metro.cl
