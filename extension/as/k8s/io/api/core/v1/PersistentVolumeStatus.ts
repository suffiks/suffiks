// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";

export class PersistentVolumeStatus {
  static encode(message: PersistentVolumeStatus, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.phase);

    writer.uint32(18);
    writer.string(message.message);

    writer.uint32(26);
    writer.string(message.reason);
  }

  static decode(reader: Reader, length: i32): PersistentVolumeStatus {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new PersistentVolumeStatus();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.phase = reader.string();
          break;

        case 2:
          message.message = reader.string();
          break;

        case 3:
          message.reason = reader.string();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  phase: string;
  message: string;
  reason: string;

  constructor(phase: string = "", message: string = "", reason: string = "") {
    this.phase = phase;
    this.message = message;
    this.reason = reason;
  }
}
