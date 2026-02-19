import { Observable } from "rxjs";
import { parseRpcError } from "./rpcError";

export type RpcTransportConfig = {
  url: string | (() => string);
  credentials?: RequestCredentials; // default: "include"
};

/**
 * The Rpc interface expected by ts-proto generated clients.
 */
export interface RpcTransport {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
  clientStreamingRequest(
    service: string,
    method: string,
    data: Observable<Uint8Array>
  ): Promise<Uint8Array>;
  serverStreamingRequest(
    service: string,
    method: string,
    data: Uint8Array
  ): Observable<Uint8Array>;
  bidirectionalStreamingRequest(
    service: string,
    method: string,
    data: Observable<Uint8Array>
  ): Observable<Uint8Array>;
}

export function createRpcTransport(config: RpcTransportConfig): RpcTransport {
  const getUrl = typeof config.url === "function" ? config.url : () => config.url as string;
  const credentials = config.credentials ?? "include";

  return {
    request(_service, method, data) {
      return fetch(getUrl(), {
        method: "POST",
        headers: {
          "Content-Type": "application/x-protobuf",
          "X-Rpc-Method": method,
        },
        credentials,
        body: data as unknown as BodyInit,
      })
        .then(async (res) => {
          if (!res.ok) {
            throw await parseRpcError(res);
          }
          return res.arrayBuffer();
        })
        .then((buffer) => {
          return new Uint8Array(buffer);
        });
    },

    clientStreamingRequest(
      _service: string,
      method: string,
      data: Observable<Uint8Array>
    ): Promise<Uint8Array> {
      return new Promise<Uint8Array>((resolve, reject) => {
        const chunks: Uint8Array[] = [];

        data.subscribe({
          next(encoded) {
            // Prepend 4-byte big-endian length prefix
            const prefix = new Uint8Array(4);
            const view = new DataView(prefix.buffer);
            view.setUint32(0, encoded.length, false);
            chunks.push(prefix, encoded);
          },
          error(err) {
            reject(err);
          },
          complete() {
            // Concatenate all prefixed chunks into a single body
            let totalLength = 0;
            for (const chunk of chunks) {
              totalLength += chunk.length;
            }
            const body = new Uint8Array(totalLength);
            let offset = 0;
            for (const chunk of chunks) {
              body.set(chunk, offset);
              offset += chunk.length;
            }

            // Send as a regular POST
            fetch(getUrl(), {
              method: "POST",
              headers: {
                "Content-Type": "application/x-protobuf",
                "X-Rpc-Method": method,
              },
              credentials,
              body: body as unknown as BodyInit,
            })
              .then(async (res) => {
                if (!res.ok) {
                  throw await parseRpcError(res);
                }
                return res.arrayBuffer();
              })
              .then((buffer) => resolve(new Uint8Array(buffer)))
              .catch(reject);
          },
        });
      });
    },

    serverStreamingRequest(
      _service: string,
      method: string,
      data: Uint8Array
    ): Observable<Uint8Array> {
      return new Observable<Uint8Array>((subscriber) => {
        let aborted = false;

        fetch(getUrl(), {
          method: "POST",
          headers: {
            "Content-Type": "application/x-protobuf",
            "X-Rpc-Method": method,
          },
          credentials,
          body: data as unknown as BodyInit,
        })
          .then(async (response) => {
            if (!response.ok) {
              throw await parseRpcError(response);
            }

            const reader = response.body?.getReader();
            if (!reader) {
              throw new Error("No response body");
            }

            let buffer = new Uint8Array(0);

            try {
              while (!aborted) {
                const { done, value } = await reader.read();

                if (done) {
                  break;
                }

                // Append new data to buffer
                const newBuffer = new Uint8Array(buffer.length + value.length);
                newBuffer.set(buffer);
                newBuffer.set(value, buffer.length);
                buffer = newBuffer;

                // Parse complete messages from buffer
                while (buffer.length >= 4) {
                  // Read 4-byte length prefix (big endian)
                  const length =
                    (buffer[0]! << 24) |
                    (buffer[1]! << 16) |
                    (buffer[2]! << 8) |
                    buffer[3]!;

                  // Check if we have the complete message
                  if (buffer.length < 4 + length) {
                    break; // Need more data
                  }

                  // Extract the message
                  const message = buffer.slice(4, 4 + length);

                  // Remove processed data from buffer
                  buffer = buffer.slice(4 + length);

                  // Emit the message
                  subscriber.next(message);
                }
              }

              subscriber.complete();
            } catch (error) {
              if (!aborted) {
                subscriber.error(error);
              }
            }
          })
          .catch((error) => {
            if (!aborted) {
              subscriber.error(error);
            }
          });

        // Cleanup function
        return () => {
          aborted = true;
        };
      });
    },

    bidirectionalStreamingRequest(
      _service: string,
      _method: string,
      _data: Observable<Uint8Array>
    ): Observable<Uint8Array> {
      throw new Error("Bidirectional streaming not implemented");
    },
  };
}
