// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { ObjectMeta } from "../../../apimachinery/pkg/apis/meta/v1/ObjectMeta";
import { NamespaceSpec } from "./NamespaceSpec";
import { NamespaceStatus } from "./NamespaceStatus";

export class Namespace {
  static encode(message: Namespace, writer: Writer): void {
    const metadata = message.metadata;
    if (metadata !== null) {
      writer.uint32(10);
      writer.fork();
      ObjectMeta.encode(metadata, writer);
      writer.ldelim();
    }

    const spec = message.spec;
    if (spec !== null) {
      writer.uint32(18);
      writer.fork();
      NamespaceSpec.encode(spec, writer);
      writer.ldelim();
    }

    const status = message.status;
    if (status !== null) {
      writer.uint32(26);
      writer.fork();
      NamespaceStatus.encode(status, writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): Namespace {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new Namespace();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.metadata = ObjectMeta.decode(reader, reader.uint32());
          break;

        case 2:
          message.spec = NamespaceSpec.decode(reader, reader.uint32());
          break;

        case 3:
          message.status = NamespaceStatus.decode(reader, reader.uint32());
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  metadata: ObjectMeta | null;
  spec: NamespaceSpec | null;
  status: NamespaceStatus | null;

  constructor(
    metadata: ObjectMeta | null = null,
    spec: NamespaceSpec | null = null,
    status: NamespaceStatus | null = null
  ) {
    this.metadata = metadata;
    this.spec = spec;
    this.status = status;
  }
}
