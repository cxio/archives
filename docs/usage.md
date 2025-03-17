# Archives Storage Service

A service for storing and retrieving documents with hash-based indexing.

## Usage

```
archives [options]
```

## Options

-h, --help      Show this help information
-v, --version   Show version information
--config FILE   Path to configuration file (default: config.json)

## API Endpoints

- POST /api/document     Upload a new document
- GET /api/document/:id  Retrieve a document by ID
- HEAD /api/document/:id Check if a document exists
- GET /api/meta/:id      Get document metadata by ID

## Configuration

The service can be configured via config.json file:
- serve_port: HTTP server port
- storage_root_path: Root directory for document storage
- default_language: Default language for metadata
- log_level: Logging level (debug, info, warn, error)

## Examples

Upload a document:
```
curl -X POST -F "file=@document.pdf" http://localhost:8080/api/document
```

Retrieve a document:
```
curl http://localhost:8080/api/document/[document_id] -o document.pdf
```
