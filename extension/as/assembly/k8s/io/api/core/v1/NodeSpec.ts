// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { Taint } from "./Taint";
import { NodeConfigSource } from "./NodeConfigSource";

export class NodeSpec {
  static encode(message: NodeSpec, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.podCIDR);

    const podCIDRs = message.podCIDRs;
    if (podCIDRs.length !== 0) {
      for (let i: i32 = 0; i < podCIDRs.length; ++i) {
        writer.uint32(58);
        writer.string(podCIDRs[i]);
      }
    }

    writer.uint32(26);
    writer.string(message.providerID);

    writer.uint32(32);
    writer.bool(message.unschedulable);

    const taints = message.taints;
    for (let i: i32 = 0; i < taints.length; ++i) {
      writer.uint32(42);
      writer.fork();
      Taint.encode(taints[i], writer);
      writer.ldelim();
    }

    const configSource = message.configSource;
    if (configSource !== null) {
      writer.uint32(50);
      writer.fork();
      NodeConfigSource.encode(configSource, writer);
      writer.ldelim();
    }

    writer.uint32(18);
    writer.string(message.externalID);
  }

  static decode(reader: Reader, length: i32): NodeSpec {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new NodeSpec();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.podCIDR = reader.string();
          break;

        case 7:
          message.podCIDRs.push(reader.string());
          break;

        case 3:
          message.providerID = reader.string();
          break;

        case 4:
          message.unschedulable = reader.bool();
          break;

        case 5:
          message.taints.push(Taint.decode(reader, reader.uint32()));
          break;

        case 6:
          message.configSource = NodeConfigSource.decode(
            reader,
            reader.uint32()
          );
          break;

        case 2:
          message.externalID = reader.string();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  podCIDR: string;
  podCIDRs: Array<string>;
  providerID: string;
  unschedulable: bool;
  taints: Array<Taint>;
  configSource: NodeConfigSource | null;
  externalID: string;

  constructor(
    podCIDR: string = "",
    podCIDRs: Array<string> = [],
    providerID: string = "",
    unschedulable: bool = false,
    taints: Array<Taint> = [],
    configSource: NodeConfigSource | null = null,
    externalID: string = ""
  ) {
    this.podCIDR = podCIDR;
    this.podCIDRs = podCIDRs;
    this.providerID = providerID;
    this.unschedulable = unschedulable;
    this.taints = taints;
    this.configSource = configSource;
    this.externalID = externalID;
  }
}