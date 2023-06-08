// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { ObjectMeta } from "../../../apimachinery/pkg/apis/meta/v1/ObjectMeta";
import { PodStatus } from "./PodStatus";

export class PodStatusResult {
  static encode(message: PodStatusResult, writer: Writer): void {
    const metadata = message.metadata;
    if (metadata !== null) {
      writer.uint32(10);
      writer.fork();
      ObjectMeta.encode(metadata, writer);
      writer.ldelim();
    }

    const status = message.status;
    if (status !== null) {
      writer.uint32(18);
      writer.fork();
      PodStatus.encode(status, writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): PodStatusResult {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new PodStatusResult();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.metadata = ObjectMeta.decode(reader, reader.uint32());
          break;

        case 2:
          message.status = PodStatus.decode(reader, reader.uint32());
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  metadata: ObjectMeta | null;
  status: PodStatus | null;

  constructor(
    metadata: ObjectMeta | null = null,
    status: PodStatus | null = null
  ) {
    this.metadata = metadata;
    this.status = status;
  }
}
