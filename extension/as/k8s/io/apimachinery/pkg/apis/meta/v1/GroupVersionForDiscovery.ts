// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";

export class GroupVersionForDiscovery {
  static encode(message: GroupVersionForDiscovery, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.groupVersion);

    writer.uint32(18);
    writer.string(message.version);
  }

  static decode(reader: Reader, length: i32): GroupVersionForDiscovery {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new GroupVersionForDiscovery();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.groupVersion = reader.string();
          break;

        case 2:
          message.version = reader.string();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  groupVersion: string;
  version: string;

  constructor(groupVersion: string = "", version: string = "") {
    this.groupVersion = groupVersion;
    this.version = version;
  }
}
