// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { Owner } from "./Owner";

export class SyncRequest {
  static encode(message: SyncRequest, writer: Writer): void {
    const owner = message.owner;
    if (owner !== null) {
      writer.uint32(10);
      writer.fork();
      Owner.encode(owner, writer);
      writer.ldelim();
    }

    writer.uint32(18);
    writer.bytes(message.spec);
  }

  static decode(reader: Reader, length: i32): SyncRequest {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new SyncRequest();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = Owner.decode(reader, reader.uint32());
          break;

        case 2:
          message.spec = reader.bytes();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  owner: Owner | null;
  spec: Uint8Array;

  constructor(
    owner: Owner | null = null,
    spec: Uint8Array = new Uint8Array(0)
  ) {
    this.owner = owner;
    this.spec = spec;
  }
}
