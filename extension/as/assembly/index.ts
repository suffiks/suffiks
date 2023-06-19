import { Protobuf, Reader } from "as-proto/assembly";
import { JSON } from "json-as";
import { EnvFrom } from "./extension/EnvFrom";
import { EnvFromType } from "./extension/EnvFromType";
import { KeyValue } from "./extension/KeyValue";
import { Owner } from "./extension/Owner";
import { ValidationError } from "./extension/ValidationError";
import { ValidationType as PBValidationType } from "./extension/ValidationType";
import { Container } from "./k8s/io/api/core/v1/Container";
import { GroupVersionResource } from "./k8s/io/apimachinery/pkg/apis/meta/v1/GroupVersionResource";
import { ClientError } from "./suffiks/clientError";
import { Host } from "./suffiks/env";

export { ValidationError, PBValidationType as ValidationType };

export namespace Suffiks {
  class Resource<T> {
    error: ClientError | null = null;
    resource: T | null = null;

    constructor(error: ClientError | null = null, resource: T | null = null) {
      this.error = error;
      this.resource = resource;
    }

    public hasError(): bool {
      return this.error != null;
    }
  }

  /**
   * AddEnv adds an environment variable to the main container.
   */
  export function addEnv(name: string, value: string): void {
    const kv = new KeyValue(name, value);
    const b = Protobuf.encode(kv, KeyValue.encode);
    Host.addEnv(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * AddEnvFrom adds an environment variable from a secret or configmap to the main container.
   */
  export function addEnvFrom(
    name: string,
    type: EnvFromType,
    optional: boolean = false
  ): void {
    const envFrom = new EnvFrom(name, optional, type);
    const b = Protobuf.encode(envFrom, EnvFrom.encode);
    Host.addEnvFrom(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * AddLabel adds a label to the current resource.
   */
  export function addLabel(name: string, value: string): void {
    const kv = new KeyValue(name, value);
    const b = Protobuf.encode(kv, KeyValue.encode);
    Host.addLabel(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * AddAnnotation adds an annotation to the current resource.
   */
  export function addAnnotation(name: string, value: string): void {
    const kv = new KeyValue(name, value);
    const b = Protobuf.encode(kv, KeyValue.encode);
    Host.addAnnotation(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * AddInitContainer adds an init container to the current resource.
   */
  export function addInitContainer(container: Container): void {
    const b = Protobuf.encode(container, Container.encode);
    Host.addInitContainer(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * AddSidecar adds a sidecar container to the current resource.
   */
  export function addSidecar(container: Container): void {
    const b = Protobuf.encode(container, Container.encode);
    Host.addSidecar(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * MergePatch merges the given patch into the current resource.
   */
  export function mergePatch(patch: string): void {
    const b = String.UTF8.encode(patch);
    const ar = changetype<Uint8Array>(b);

    Host.mergePatch(ar.dataStart as u32, ar.byteLength as u32);
  }

  /**
   * GetOwner returns the owner of the current resource.
   *
   * @returns the owner of the current resource
   */
  export function getOwner(): Owner {
    const ptrAndSize = Host.getOwner() as u32;
    return decode<Owner>(ptrAndSize, Owner.decode);
  }

  /**
   * Get the spec defined for the extension.
   *
   * @returns the spec defined for the extension
   */
  export function getSpec<T>(): T {
    const ptrAndSize = Host.getSpec() as u32;
    const b = getArray(ptrAndSize);

    const s = String.UTF8.decode(b.buffer);
    return JSON.parse<T>(s);
  }

  /**
   * GetOld returns the currently applied resource during a validation webhook request.
   *
   * @returns the spec defined for the extension
   */
  export function getOld<T>(): T {
    const ptrAndSize = Host.getOld() as u32;
    const b = getArray(ptrAndSize);

    const s = String.UTF8.decode(b.buffer);
    return JSON.parse<T>(s);
  }

  /**
   * Report a validation error.
   */
  export function validationError(
    path: string,
    detail: string,
    value: string
  ): void {
    const err = new ValidationError(path, detail, value);
    const b = Protobuf.encode(err, ValidationError.encode);

    Host.validationError(b.dataStart as u32, b.buffer.byteLength as u32);
  }

  /**
   * Get a resource
   */
  export function getResource<T>(
    group: string,
    version: string,
    resource: string,
    name: string
  ): Resource<T> {
    const gvr = new GroupVersionResource(group, version, resource);
    const b = Protobuf.encode(gvr, GroupVersionResource.encode);
    const nameb = String.UTF8.encode(name);
    const namePtr = changetype<u32>(nameb);

    const ptrAndSize = Host.getResource(
      b.dataStart as u32,
      b.buffer.byteLength as u32,
      namePtr as u32,
      nameb.byteLength as u32
    );
    if (ptrAndSize < 20) {
      return new Resource(new ClientError(ptrAndSize));
    }

    const ar = getArray(ptrAndSize);
    const s = String.UTF8.decode(ar.buffer);
    return new Resource(null, JSON.parse<T>(s));
  }

  /**
   * Create a new resource.
   */
  export function createResource<T>(
    group: string,
    version: string,
    resource: string,
    res: string
  ): Resource<T> {
    const gvr = new GroupVersionResource(group, version, resource);
    const b = Protobuf.encode(gvr, GroupVersionResource.encode);

    const resb = String.UTF8.encode(res);
    const resPtr = changetype<u32>(resb);

    const ptrSizeOrErr = Host.createResource(
      b.dataStart as u32,
      b.buffer.byteLength as u32,
      resPtr as u32,
      resb.byteLength as u32
    );
    if (ptrSizeOrErr < 20) {
      return new Resource(new ClientError(ptrSizeOrErr));
    }

    const ar = getArray(ptrSizeOrErr);
    const s = String.UTF8.decode(ar.buffer);
    return new Resource(null, JSON.parse<T>(s));
  }

  /**
   * Update an existing resource.
   */
  export function updateResource<T>(
    group: string,
    version: string,
    resource: string,
    res: string
  ): Resource<T> {
    const gvr = new GroupVersionResource(group, version, resource);
    const b = Protobuf.encode(gvr, GroupVersionResource.encode);

    const resb = String.UTF8.encode(res);
    const resPtr = changetype<u32>(resb);

    const ptrSizeOrErr = Host.updateResource(
      b.dataStart as u32,
      b.buffer.byteLength as u32,
      resPtr as u32,
      resb.byteLength as u32
    );
    if (ptrSizeOrErr < 20) {
      return new Resource(new ClientError(ptrSizeOrErr));
    }

    const ar = getArray(ptrSizeOrErr);
    const s = String.UTF8.decode(ar.buffer);
    return new Resource(null, JSON.parse<T>(s));
  }

  /**
   * Delete an existing resource.
   */
  export function deleteResource(
    group: string,
    version: string,
    resource: string,
    name: string
  ): ClientError | null {
    const gvr = new GroupVersionResource(group, version, resource);
    const b = Protobuf.encode(gvr, GroupVersionResource.encode);
    const nameb = String.UTF8.encode(name);
    const namePtr = changetype<u32>(nameb);

    const err = Host.deleteResource(
      b.dataStart as u32,
      b.buffer.byteLength as u32,
      namePtr as u32,
      nameb.byteLength as u32
    );
    if (err > 0) {
      return new ClientError(err);
    }

    return null;
  }

  /**
   * Which type of validation is being performed.
   *
   * @param vt int32 value of the validation type
   * @returns
   */
  export function validationType(vt: i32): PBValidationType {
    switch (vt) {
      case 1:
        return PBValidationType.UPDATE;
      case 2:
        return PBValidationType.DELETE;
    }

    return PBValidationType.CREATE;
  }

  /**
   * Encode a class to JSON and return the information required for Suffiks to
   * decode it.
   * @returns
   */
  export function defaultingResponse<T>(obj: T): u32 {
    const s = JSON.stringify(obj);
    return stringToPtr(s);
  }

  function stringToPtr(s: string): u32 {
    const enc = String.UTF8.encode(s);
    const ptr = changetype<u32>(enc);
    const size = enc.byteLength;
    // Shift the combined pointer and size by 16 bits to get a single u64 value
    // with the pointer in the upper 16 bits and the size in the lower 16 bits.
    return ((ptr as u32) << 16) | (size as u32);
  }

  function decode<T>(
    ptrAndSize: u32,
    decoder: (reader: Reader, length: i32) => T
  ): T {
    const b = getArray(ptrAndSize);
    return Protobuf.decode<T>(b, decoder);
  }

  function getArray(ptrAndSize: u32): Uint8Array {
    var size: u32 = ptrAndSize & 0xffff;
    // Shift the combined value to the right by 16 bits to get the pointer address
    var ptr: u32 = ptrAndSize >>> 16;

    const b = new Uint8Array(size);
    for (let i = u32(0); i < size; i++) {
      b[i] = load<u8>(ptr + i);
    }

    heap.free(ptr);
    return b;
  }
}
