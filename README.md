# AI Tickets

Household issue tickets that classify and auto-resolve. A report like “the language of this movie is wrong” becomes a `replace_media` resolution peers can turn into a `media.movie.requested` event. This module does not embed request-media logic.

## Ports

| Service | Default |
|---------|---------|
| gRPC | `127.0.0.1:9766` |
| HTTP | `127.0.0.1:9767` |

## HTTP

| Method | Path |
|--------|------|
| POST | `/v1/tickets` |
| GET | `/v1/tickets` |
| POST | `/v1/tickets/{id}/resolve` |

Reporter identities are PII. Tickets are operational records with a delete path (in-memory until restart).
