// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { LoadBalancerStatus } from "./LoadBalancerStatus";
import { Condition } from "../../../apimachinery/pkg/apis/meta/v1/Condition";

export class ServiceStatus {
  static encode(message: ServiceStatus, writer: Writer): void {
    const loadBalancer = message.loadBalancer;
    if (loadBalancer !== null) {
      writer.uint32(10);
      writer.fork();
      LoadBalancerStatus.encode(loadBalancer, writer);
      writer.ldelim();
    }

    const conditions = message.conditions;
    for (let i: i32 = 0; i < conditions.length; ++i) {
      writer.uint32(18);
      writer.fork();
      Condition.encode(conditions[i], writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): ServiceStatus {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new ServiceStatus();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.loadBalancer = LoadBalancerStatus.decode(
            reader,
            reader.uint32()
          );
          break;

        case 2:
          message.conditions.push(Condition.decode(reader, reader.uint32()));
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  loadBalancer: LoadBalancerStatus | null;
  conditions: Array<Condition>;

  constructor(
    loadBalancer: LoadBalancerStatus | null = null,
    conditions: Array<Condition> = []
  ) {
    this.loadBalancer = loadBalancer;
    this.conditions = conditions;
  }
}