// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { ClientIPConfig } from "./ClientIPConfig";

export class SessionAffinityConfig {
  static encode(message: SessionAffinityConfig, writer: Writer): void {
    const clientIP = message.clientIP;
    if (clientIP !== null) {
      writer.uint32(10);
      writer.fork();
      ClientIPConfig.encode(clientIP, writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): SessionAffinityConfig {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new SessionAffinityConfig();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.clientIP = ClientIPConfig.decode(reader, reader.uint32());
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  clientIP: ClientIPConfig | null;

  constructor(clientIP: ClientIPConfig | null = null) {
    this.clientIP = clientIP;
  }
}