# simpleemailapi

## 0.2.0

### Minor Changes

- **Non-blocking Event Streams**: Moved `client.onReceive()` execution to a dedicated background worker thread (`worker_threads`).
- Ensures the main event loop remains unblocked during high-volume email ingestion.
- Automatic reconnection with exponential backoff.
- Seamless background batch processing.

## 0.1.0

### Features

- Initial release
- `createClient()` - Create a typed Email API client
- `client.send()` - Send transactional emails
- `client.onReceive()` - Stream email events with typed callbacks
- `client.email` - Full EmailService client access
- `client.domains` - Full DomainService client access
- All proto-generated types exported for full type safety
