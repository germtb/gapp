import { BinaryReader } from "@bufbuild/protobuf/wire";

// Preloaded data structure embedded in HTML (gzip-compressed, base64-encoded protobuf bytes)
export type PreloadedData = {
  [method: string]: {
    requestBytes: string;
    responseBytes: string;
  };
};

declare global {
  interface Window {
    __PRELOADED__?: PreloadedData;
    __PRELOAD_TIMESTAMP__?: number;
  }
}

// RPC declaration for route preloading
export type RpcDeclaration = {
  method: string;
  params?: Record<string, string>;
};

export type DecoderMap = Record<
  string,
  (reader: BinaryReader) => unknown
>;

/**
 * Decode base64 string to Uint8Array
 */
function base64ToBytes(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

/**
 * Decompress gzipped bytes using DecompressionStream API
 */
async function decompressGzip(compressed: Uint8Array): Promise<Uint8Array> {
  const ds = new DecompressionStream("gzip");
  const writer = ds.writable.getWriter();
  writer.write(compressed as unknown as BufferSource);
  writer.close();

  const reader = ds.readable.getReader();
  const chunks: Uint8Array[] = [];
  let totalLength = 0;

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    totalLength += value.length;
  }

  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of chunks) {
    result.set(chunk, offset);
    offset += chunk.length;
  }
  return result;
}

/**
 * Decode and dispatch all preloaded RPCs.
 * Returns array of { method, request, response } for dispatching to stores.
 * Data is gzip compressed, so decompression is async.
 */
export async function decodeAllPreloaded(
  requestDecoders: DecoderMap,
  responseDecoders: DecoderMap
): Promise<
  Array<{
    method: string;
    request: unknown;
    response: unknown;
  }>
> {
  const preloaded = window.__PRELOADED__;
  if (!preloaded) {
    return [];
  }

  const results: Array<{
    method: string;
    request: unknown;
    response: unknown;
  }> = [];

  for (const [method, { requestBytes, responseBytes }] of Object.entries(
    preloaded
  )) {
    const reqDecoder = requestDecoders[method];
    const resDecoder = responseDecoders[method];
    if (!resDecoder) {
      console.warn(`[Preload] No decoder for method: ${method}`);
      continue;
    }

    try {
      // Decode request (may be empty for some RPCs)
      let request: unknown = {};
      if (requestBytes && reqDecoder) {
        const compressedReq = base64ToBytes(requestBytes);
        const reqBytes = await decompressGzip(compressedReq);
        request = reqDecoder(new BinaryReader(reqBytes));
      }

      // Decode response (decompress gzipped data)
      const compressedRes = base64ToBytes(responseBytes);
      const resBytes = await decompressGzip(compressedRes);
      const response = resDecoder(new BinaryReader(resBytes));

      results.push({ method, request, response });
    } catch (err) {
      console.error(`[Preload] Failed to decode ${method}:`, err);
    }
  }

  // Clear preloaded data
  delete window.__PRELOADED__;

  return results;
}
