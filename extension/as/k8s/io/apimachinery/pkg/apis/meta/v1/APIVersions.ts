// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { ServerAddressByClientCIDR } from "./ServerAddressByClientCIDR";

export class APIVersions {
  static encode(message: APIVersions, writer: Writer): void {
    const versions = message.versions;
    if (versions.length !== 0) {
      for (let i: i32 = 0; i < versions.length; ++i) {
        writer.uint32(10);
        writer.string(versions[i]);
      }
    }

    const serverAddressByClientCIDRs = message.serverAddressByClientCIDRs;
    for (let i: i32 = 0; i < serverAddressByClientCIDRs.length; ++i) {
      writer.uint32(18);
      writer.fork();
      ServerAddressByClientCIDR.encode(serverAddressByClientCIDRs[i], writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): APIVersions {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new APIVersions();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.versions.push(reader.string());
          break;

        case 2:
          message.serverAddressByClientCIDRs.push(
            ServerAddressByClientCIDR.decode(reader, reader.uint32())
          );
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  versions: Array<string>;
  serverAddressByClientCIDRs: Array<ServerAddressByClientCIDR>;

  constructor(
    versions: Array<string> = [],
    serverAddressByClientCIDRs: Array<ServerAddressByClientCIDR> = []
  ) {
    this.versions = versions;
    this.serverAddressByClientCIDRs = serverAddressByClientCIDRs;
  }
}
