// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { ObjectReference } from "./ObjectReference";

export class EndpointAddress {
  static encode(message: EndpointAddress, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.ip);

    writer.uint32(26);
    writer.string(message.hostname);

    writer.uint32(34);
    writer.string(message.nodeName);

    const targetRef = message.targetRef;
    if (targetRef !== null) {
      writer.uint32(18);
      writer.fork();
      ObjectReference.encode(targetRef, writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): EndpointAddress {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new EndpointAddress();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.ip = reader.string();
          break;

        case 3:
          message.hostname = reader.string();
          break;

        case 4:
          message.nodeName = reader.string();
          break;

        case 2:
          message.targetRef = ObjectReference.decode(reader, reader.uint32());
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  ip: string;
  hostname: string;
  nodeName: string;
  targetRef: ObjectReference | null;

  constructor(
    ip: string = "",
    hostname: string = "",
    nodeName: string = "",
    targetRef: ObjectReference | null = null
  ) {
    this.ip = ip;
    this.hostname = hostname;
    this.nodeName = nodeName;
    this.targetRef = targetRef;
  }
}