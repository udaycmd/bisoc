## Bisoc

⚠️ **Experimental library that implements the RFC 6455 WebSocket protocol in Go.**

### Features

- Implements the core [RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455) WebSocket protocol specification.
- Zero external dependencies (uses only the Go standard library).
- Supports data message fragmentation and continuation frames.
- Automatic frame masking (client-side) and unmasking (server-side).
- Handles control frames (`Ping`, `Pong`, `Close`).

### Limitations & Drawbacks

- **No Extensions**: This library does not implement any WebSocket extensions (such as `permessage-deflate` for compression).
- **Not for Production**: The codebase is intended purely for educational purposes and experimental use. It has not been optimized for high concurrency or production security standards.
- **UTF-8 Validation**: UTF-8 validation for text messages is only performed at the complete message boundary, not at the individual frame level for fragmented messages.

### Autobahn Test Suite

This project includes the [Autobahn Testsuite](https://github.com/crossbario/autobahn-testsuite) in the `test/autobahn` directory to verify its compliance with the WebSocket protocol.

#### How to Run Locally

You can run the full test suite locally against the provided echo server using Docker.

1. **Start the Echo Server**
   Open a terminal in the project root and run the test echo server on port 8080:

   ```bash
   go run test/autobahn/test_echo_server.go
   ```

2. **Run the Autobahn Testsuite**
   Open another terminal and use Docker to run the test suite:

   ```bash
   sudo docker run -it --rm \
    --network host \
    -v "${PWD}/test/autobahn/config:/config" \
    -v "${PWD}/test/autobahn/reports:/reports" \
    crossbario/autobahn-testsuite \
    wstest -m fuzzingclient -s /config/fuzzingclient.json
   ```

3. **View the Results**
   Once the test suite finishes, it will generate an HTML report in the `test/autobahn/reports` directory. You can open `test/autobahn/reports/index.html` in your browser to view the detailed compliance results.
