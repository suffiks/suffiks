// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { NamespaceCondition } from "./NamespaceCondition";

export class NamespaceStatus {
  static encode(message: NamespaceStatus, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.phase);

    const conditions = message.conditions;
    for (let i: i32 = 0; i < conditions.length; ++i) {
      writer.uint32(18);
      writer.fork();
      NamespaceCondition.encode(conditions[i], writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): NamespaceStatus {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new NamespaceStatus();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.phase = reader.string();
          break;

        case 2:
          message.conditions.push(
            NamespaceCondition.decode(reader, reader.uint32())
          );
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  phase: string;
  conditions: Array<NamespaceCondition>;

  constructor(phase: string = "", conditions: Array<NamespaceCondition> = []) {
    this.phase = phase;
    this.conditions = conditions;
  }
}