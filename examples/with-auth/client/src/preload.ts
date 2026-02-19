import { decodeAllPreloaded, type DecoderMap } from "@gap/client";
import {
  GetItemsRequest,
  GetItemsResponse,
} from "./generated/service";

const requestDecoders: DecoderMap = {
  GetItems: (reader) => GetItemsRequest.decode(reader),
};

const responseDecoders: DecoderMap = {
  GetItems: (reader) => GetItemsResponse.decode(reader),
};

export function decodePreloaded() {
  return decodeAllPreloaded(requestDecoders, responseDecoders);
}
