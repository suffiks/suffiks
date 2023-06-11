// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";

export class FlockerVolumeSource {
  static encode(message: FlockerVolumeSource, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.datasetName);

    writer.uint32(18);
    writer.string(message.datasetUUID);
  }

  static decode(reader: Reader, length: i32): FlockerVolumeSource {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new FlockerVolumeSource();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.datasetName = reader.string();
          break;

        case 2:
          message.datasetUUID = reader.string();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  datasetName: string;
  datasetUUID: string;

  constructor(datasetName: string = "", datasetUUID: string = "") {
    this.datasetName = datasetName;
    this.datasetUUID = datasetUUID;
  }
}