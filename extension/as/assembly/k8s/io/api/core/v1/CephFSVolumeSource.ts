// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { LocalObjectReference } from "./LocalObjectReference";

export class CephFSVolumeSource {
  static encode(message: CephFSVolumeSource, writer: Writer): void {
    const monitors = message.monitors;
    if (monitors.length !== 0) {
      for (let i: i32 = 0; i < monitors.length; ++i) {
        writer.uint32(10);
        writer.string(monitors[i]);
      }
    }

    writer.uint32(18);
    writer.string(message.path);

    writer.uint32(26);
    writer.string(message.user);

    writer.uint32(34);
    writer.string(message.secretFile);

    const secretRef = message.secretRef;
    if (secretRef !== null) {
      writer.uint32(42);
      writer.fork();
      LocalObjectReference.encode(secretRef, writer);
      writer.ldelim();
    }

    writer.uint32(48);
    writer.bool(message.readOnly);
  }

  static decode(reader: Reader, length: i32): CephFSVolumeSource {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new CephFSVolumeSource();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.monitors.push(reader.string());
          break;

        case 2:
          message.path = reader.string();
          break;

        case 3:
          message.user = reader.string();
          break;

        case 4:
          message.secretFile = reader.string();
          break;

        case 5:
          message.secretRef = LocalObjectReference.decode(
            reader,
            reader.uint32()
          );
          break;

        case 6:
          message.readOnly = reader.bool();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  monitors: Array<string>;
  path: string;
  user: string;
  secretFile: string;
  secretRef: LocalObjectReference | null;
  readOnly: bool;

  constructor(
    monitors: Array<string> = [],
    path: string = "",
    user: string = "",
    secretFile: string = "",
    secretRef: LocalObjectReference | null = null,
    readOnly: bool = false
  ) {
    this.monitors = monitors;
    this.path = path;
    this.user = user;
    this.secretFile = secretFile;
    this.secretRef = secretRef;
    this.readOnly = readOnly;
  }
}