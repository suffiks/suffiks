// Code generated by protoc-gen-as. DO NOT EDIT.
// Versions:
//   protoc-gen-as v1.3.0
//   protoc        v3.20.1

import { Writer, Reader } from "as-proto/assembly";
import { ContainerPort } from "./ContainerPort";
import { EnvFromSource } from "./EnvFromSource";
import { EnvVar } from "./EnvVar";
import { ResourceRequirements } from "./ResourceRequirements";
import { VolumeMount } from "./VolumeMount";
import { VolumeDevice } from "./VolumeDevice";
import { Probe } from "./Probe";
import { Lifecycle } from "./Lifecycle";
import { SecurityContext } from "./SecurityContext";

export class Container {
  static encode(message: Container, writer: Writer): void {
    writer.uint32(10);
    writer.string(message.name);

    writer.uint32(18);
    writer.string(message.image);

    const command = message.command;
    if (command.length !== 0) {
      for (let i: i32 = 0; i < command.length; ++i) {
        writer.uint32(26);
        writer.string(command[i]);
      }
    }

    const args = message.args;
    if (args.length !== 0) {
      for (let i: i32 = 0; i < args.length; ++i) {
        writer.uint32(34);
        writer.string(args[i]);
      }
    }

    writer.uint32(42);
    writer.string(message.workingDir);

    const ports = message.ports;
    for (let i: i32 = 0; i < ports.length; ++i) {
      writer.uint32(50);
      writer.fork();
      ContainerPort.encode(ports[i], writer);
      writer.ldelim();
    }

    const envFrom = message.envFrom;
    for (let i: i32 = 0; i < envFrom.length; ++i) {
      writer.uint32(154);
      writer.fork();
      EnvFromSource.encode(envFrom[i], writer);
      writer.ldelim();
    }

    const env = message.env;
    for (let i: i32 = 0; i < env.length; ++i) {
      writer.uint32(58);
      writer.fork();
      EnvVar.encode(env[i], writer);
      writer.ldelim();
    }

    const resources = message.resources;
    if (resources !== null) {
      writer.uint32(66);
      writer.fork();
      ResourceRequirements.encode(resources, writer);
      writer.ldelim();
    }

    const volumeMounts = message.volumeMounts;
    for (let i: i32 = 0; i < volumeMounts.length; ++i) {
      writer.uint32(74);
      writer.fork();
      VolumeMount.encode(volumeMounts[i], writer);
      writer.ldelim();
    }

    const volumeDevices = message.volumeDevices;
    for (let i: i32 = 0; i < volumeDevices.length; ++i) {
      writer.uint32(170);
      writer.fork();
      VolumeDevice.encode(volumeDevices[i], writer);
      writer.ldelim();
    }

    const livenessProbe = message.livenessProbe;
    if (livenessProbe !== null) {
      writer.uint32(82);
      writer.fork();
      Probe.encode(livenessProbe, writer);
      writer.ldelim();
    }

    const readinessProbe = message.readinessProbe;
    if (readinessProbe !== null) {
      writer.uint32(90);
      writer.fork();
      Probe.encode(readinessProbe, writer);
      writer.ldelim();
    }

    const startupProbe = message.startupProbe;
    if (startupProbe !== null) {
      writer.uint32(178);
      writer.fork();
      Probe.encode(startupProbe, writer);
      writer.ldelim();
    }

    const lifecycle = message.lifecycle;
    if (lifecycle !== null) {
      writer.uint32(98);
      writer.fork();
      Lifecycle.encode(lifecycle, writer);
      writer.ldelim();
    }

    writer.uint32(106);
    writer.string(message.terminationMessagePath);

    writer.uint32(162);
    writer.string(message.terminationMessagePolicy);

    writer.uint32(114);
    writer.string(message.imagePullPolicy);

    const securityContext = message.securityContext;
    if (securityContext !== null) {
      writer.uint32(122);
      writer.fork();
      SecurityContext.encode(securityContext, writer);
      writer.ldelim();
    }

    writer.uint32(128);
    writer.bool(message.stdin);

    writer.uint32(136);
    writer.bool(message.stdinOnce);

    writer.uint32(144);
    writer.bool(message.tty);
  }

  static decode(reader: Reader, length: i32): Container {
    const end: usize = length < 0 ? reader.end : reader.ptr + length;
    const message = new Container();

    while (reader.ptr < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;

        case 2:
          message.image = reader.string();
          break;

        case 3:
          message.command.push(reader.string());
          break;

        case 4:
          message.args.push(reader.string());
          break;

        case 5:
          message.workingDir = reader.string();
          break;

        case 6:
          message.ports.push(ContainerPort.decode(reader, reader.uint32()));
          break;

        case 19:
          message.envFrom.push(EnvFromSource.decode(reader, reader.uint32()));
          break;

        case 7:
          message.env.push(EnvVar.decode(reader, reader.uint32()));
          break;

        case 8:
          message.resources = ResourceRequirements.decode(
            reader,
            reader.uint32()
          );
          break;

        case 9:
          message.volumeMounts.push(
            VolumeMount.decode(reader, reader.uint32())
          );
          break;

        case 21:
          message.volumeDevices.push(
            VolumeDevice.decode(reader, reader.uint32())
          );
          break;

        case 10:
          message.livenessProbe = Probe.decode(reader, reader.uint32());
          break;

        case 11:
          message.readinessProbe = Probe.decode(reader, reader.uint32());
          break;

        case 22:
          message.startupProbe = Probe.decode(reader, reader.uint32());
          break;

        case 12:
          message.lifecycle = Lifecycle.decode(reader, reader.uint32());
          break;

        case 13:
          message.terminationMessagePath = reader.string();
          break;

        case 20:
          message.terminationMessagePolicy = reader.string();
          break;

        case 14:
          message.imagePullPolicy = reader.string();
          break;

        case 15:
          message.securityContext = SecurityContext.decode(
            reader,
            reader.uint32()
          );
          break;

        case 16:
          message.stdin = reader.bool();
          break;

        case 17:
          message.stdinOnce = reader.bool();
          break;

        case 18:
          message.tty = reader.bool();
          break;

        default:
          reader.skipType(tag & 7);
          break;
      }
    }

    return message;
  }

  name: string;
  image: string;
  command: Array<string>;
  args: Array<string>;
  workingDir: string;
  ports: Array<ContainerPort>;
  envFrom: Array<EnvFromSource>;
  env: Array<EnvVar>;
  resources: ResourceRequirements | null;
  volumeMounts: Array<VolumeMount>;
  volumeDevices: Array<VolumeDevice>;
  livenessProbe: Probe | null;
  readinessProbe: Probe | null;
  startupProbe: Probe | null;
  lifecycle: Lifecycle | null;
  terminationMessagePath: string;
  terminationMessagePolicy: string;
  imagePullPolicy: string;
  securityContext: SecurityContext | null;
  stdin: bool;
  stdinOnce: bool;
  tty: bool;

  constructor(
    name: string = "",
    image: string = "",
    command: Array<string> = [],
    args: Array<string> = [],
    workingDir: string = "",
    ports: Array<ContainerPort> = [],
    envFrom: Array<EnvFromSource> = [],
    env: Array<EnvVar> = [],
    resources: ResourceRequirements | null = null,
    volumeMounts: Array<VolumeMount> = [],
    volumeDevices: Array<VolumeDevice> = [],
    livenessProbe: Probe | null = null,
    readinessProbe: Probe | null = null,
    startupProbe: Probe | null = null,
    lifecycle: Lifecycle | null = null,
    terminationMessagePath: string = "",
    terminationMessagePolicy: string = "",
    imagePullPolicy: string = "",
    securityContext: SecurityContext | null = null,
    stdin: bool = false,
    stdinOnce: bool = false,
    tty: bool = false
  ) {
    this.name = name;
    this.image = image;
    this.command = command;
    this.args = args;
    this.workingDir = workingDir;
    this.ports = ports;
    this.envFrom = envFrom;
    this.env = env;
    this.resources = resources;
    this.volumeMounts = volumeMounts;
    this.volumeDevices = volumeDevices;
    this.livenessProbe = livenessProbe;
    this.readinessProbe = readinessProbe;
    this.startupProbe = startupProbe;
    this.lifecycle = lifecycle;
    this.terminationMessagePath = terminationMessagePath;
    this.terminationMessagePolicy = terminationMessagePolicy;
    this.imagePullPolicy = imagePullPolicy;
    this.securityContext = securityContext;
    this.stdin = stdin;
    this.stdinOnce = stdinOnce;
    this.tty = tty;
  }
}