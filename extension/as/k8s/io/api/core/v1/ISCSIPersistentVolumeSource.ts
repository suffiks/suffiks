// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { SecretReference } from "./SecretReference";

export class ISCSIPersistentVolumeSource {
  static encode(message: ISCSIPersistentVolumeSource, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.targetPortal);

    writer.uint32(18);
    writer.string(message.iqn);

    writer.uint32(24);
    writer.int32(message.lun);

    writer.uint32(34);
    writer.string(message.iscsiInterface);

    writer.uint32(42);
    writer.string(message.fsType);

    writer.uint32(48);
    writer.bool(message.readOnly);

    const portals = message.portals;
    if (portals.length !== 0) {
      for (let i: i32 = 0; i < portals.length; ++i) {
        writer.uint32(58);
        writer.string(portals[i]);
      }
    }

    writer.uint32(64);
    writer.bool(message.chapAuthDiscovery);

    writer.uint32(88);
    writer.bool(message.chapAuthSession);

    const secretRef = message.secretRef;
    if (secretRef !== null) {
      writer.uint32(82);
      writer.fork();
      SecretReference.encode(secretRef, writer);
      writer.ldelim();
    }

    writer.uint32(98);
    writer.string(message.initiatorName);
  }

  static decode(reader: Reader, length: i32): ISCSIPersistentVolumeSource {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new ISCSIPersistentVolumeSource();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.targetPortal = reader.string();
          break;

        case 2:
          message.iqn = reader.string();
          break;

        case 3:
          message.lun = reader.int32();
          break;

        case 4:
          message.iscsiInterface = reader.string();
          break;

        case 5:
          message.fsType = reader.string();
          break;

        case 6:
          message.readOnly = reader.bool();
          break;

        case 7:
          message.portals.push(reader.string());
          break;

        case 8:
          message.chapAuthDiscovery = reader.bool();
          break;

        case 11:
          message.chapAuthSession = reader.bool();
          break;

        case 10:
          message.secretRef = SecretReference.decode(reader, reader.uint32());
          break;

        case 12:
          message.initiatorName = reader.string();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  targetPortal: string;
  iqn: string;
  lun: i32;
  iscsiInterface: string;
  fsType: string;
  readOnly: bool;
  portals: Array<string>;
  chapAuthDiscovery: bool;
  chapAuthSession: bool;
  secretRef: SecretReference | null;
  initiatorName: string;

  constructor(
    targetPortal: string = "",
    iqn: string = "",
    lun: i32 = 0,
    iscsiInterface: string = "",
    fsType: string = "",
    readOnly: bool = false,
    portals: Array<string> = [],
    chapAuthDiscovery: bool = false,
    chapAuthSession: bool = false,
    secretRef: SecretReference | null = null,
    initiatorName: string = ""
  ) {
    this.targetPortal = targetPortal;
    this.iqn = iqn;
    this.lun = lun;
    this.iscsiInterface = iscsiInterface;
    this.fsType = fsType;
    this.readOnly = readOnly;
    this.portals = portals;
    this.chapAuthDiscovery = chapAuthDiscovery;
    this.chapAuthSession = chapAuthSession;
    this.secretRef = secretRef;
    this.initiatorName = initiatorName;
  }
}
