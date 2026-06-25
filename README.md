# Arkivy API

This is the back-end of the project arkivy

## Tech Stack

| Tool | Version |
|------|---------|
| Go | 1.25.0 |
| Gin | v1.12.0 |
| MongoDB Driver | v2.5.1 |
| Docker | — |

## Structure
**Commander (Escritura)** : Debe realizar una acción o mutar el estado. En Go, un comando debe devolver un error o nil, pero nunca un valor de dato

**Consulta (Lectura)** : Debe responder a una pregunta sin modificar el estado. En Go, una consulta debe devolver un valor (y un error opcional), pero no debe tener efectos secundarios en la base de datos o el objeto

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) running

## Getting Started

```powershell
.\start-project.ps1
```

The API will be available at `http://localhost:9090`.

Press `Ctrl+C` to stop all containers.

### Swagger UI

Interactive docs available at `http://localhost:9090/swagger/index.html`.
