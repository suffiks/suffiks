// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";

export class CreateOptions {
  static encode(message: CreateOptions, writer: Writer): void {
    const dryRun = message.dryRun;
    if (dryRun.length !== 0) {
      for (let i: i32 = 0; i < dryRun.length; ++i) {
        writer.uint32(10);
        writer.string(dryRun[i]);
      }
    }

    writer.uint32(26);
    writer.string(message.fieldManager);

    writer.uint32(34);
    writer.string(message.fieldValidation);
  }

  static decode(reader: Reader, length: i32): CreateOptions {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new CreateOptions();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.dryRun.push(reader.string());
          break;

        case 3:
          message.fieldManager = reader.string();
          break;

        case 4:
          message.fieldValidation = reader.string();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  dryRun: Array<string>;
  fieldManager: string;
  fieldValidation: string;

  constructor(
    dryRun: Array<string> = [],
    fieldManager: string = "",
    fieldValidation: string = ""
  ) {
    this.dryRun = dryRun;
    this.fieldManager = fieldManager;
    this.fieldValidation = fieldValidation;
  }
}