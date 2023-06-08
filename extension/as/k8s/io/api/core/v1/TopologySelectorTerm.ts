// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { TopologySelectorLabelRequirement } from "./TopologySelectorLabelRequirement";

export class TopologySelectorTerm {
  static encode(message: TopologySelectorTerm, writer: Writer): void {
    const matchLabelExpressions = message.matchLabelExpressions;
    for (let i: i32 = 0; i < matchLabelExpressions.length; ++i) {
      writer.uint32(10);
      writer.fork();
      TopologySelectorLabelRequirement.encode(matchLabelExpressions[i], writer);
      writer.ldelim();
    }
  }

  static decode(reader: Reader, length: i32): TopologySelectorTerm {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new TopologySelectorTerm();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.matchLabelExpressions.push(
            TopologySelectorLabelRequirement.decode(reader, reader.uint32())
          );
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  matchLabelExpressions: Array<TopologySelectorLabelRequirement>;

  constructor(
    matchLabelExpressions: Array<TopologySelectorLabelRequirement> = []
  ) {
    this.matchLabelExpressions = matchLabelExpressions;
  }
}
