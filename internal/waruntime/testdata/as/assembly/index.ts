// The entry file of your WebAssembly module.

import { Suffiks, ValidationType } from "@suffiks/suffiks-as/assembly/index";
import { JSON } from "json-as";
import {
  HTTPIngressPath,
  HTTPIngressRuleValue,
  IngressBackend,
  IngressRule,
  IngressServiceBackend,
  IngressSpec,
  Ingress as K8sIngress,
  ObjectReference,
  ServiceBackendPort,
} from "./k8s";
export * from "@suffiks/suffiks-as/assembly/suffiks/memory";

const ingressClass = "nginx";

// Global variables to prevent issues with closures not working when accessing variables
// from outside the closure.
let name!: string;
let number = 0;

// @ts-ignore decorators are valid here
@json
class Ingress {
  host!: string;
  paths!: string[] | null;

  getPaths(): string[] {
    if (!this.paths) {
      return [];
    }
    return this.paths as string[];
  }
}

// @ts-ignore decorators are valid here
@json
class Extension {
  ingresses!: Ingress[];
}

// @ts-ignore decorators are valid here
@json
class SpecWrapper {
  spec!: Extension;
}

export function Validate(vt: i32): void {
  if (Suffiks.validationType(vt) == ValidationType.DELETE) {
    return;
  }

  const ext = Suffiks.getSpec<Extension>();

  ext.ingresses.forEach((ingress, i) => {
    number = i;
    if (!validateHost(ingress.host)) {
      Suffiks.validationError(
        "ingresses[" + i.toString() + "].host",
        "is either invalid or not accepted",
        ingress.host
      );
    }

    if (ingress.getPaths().length > 0) {
      ingress.getPaths().forEach((path, j) => {
        if (!path.startsWith("/")) {
          Suffiks.validationError(
            "ingresses[" + number.toString() + "].paths[" + j.toString() + "]",
            "must start with a slash",
            path
          );
        }
      });
    }
  });
}

export function Defaulting(): u64 {
  const ext = Suffiks.getSpec<Extension>();

  ext.ingresses.forEach((ingress, i) => {
    if (ingress.getPaths().length == 0) {
      ingress.paths = ["/"];
    }
  });

  const spec = new SpecWrapper();
  spec.spec = ext;
  const res = Suffiks.defaultingResponse(spec);
  console.log("Result: " + JSON.stringify(spec));
  return res;
}

export function Sync(): void {
  const ext = Suffiks.getSpec<Extension>();

  console.log("Add label");
  Suffiks.addLabel("is-wasm-controlled", "true");
  console.log("Added label");

  const owner = Suffiks.getOwner();
  name = owner.name;

  const spec = new K8sIngress();
  spec.apiVersion = "networking.k8s.io/v1";
  spec.kind = "Ingress";
  spec.metadata = new ObjectReference();
  spec.spec = new IngressSpec();
  spec.metadata.name = owner.name;
  spec.metadata.namespace = owner.namespace;
  spec.metadata.ownerReferences = [
    {
      apiVersion: owner.apiVersion,
      kind: owner.kind,
      name: owner.name,
      uid: owner.uid,
      controller: true,
    },
  ];

  spec.spec.ingressClassName = ingressClass;

  spec.spec.rules = ext.ingresses.map<IngressRule>((ingress): IngressRule => {
    const rule = new IngressRule();
    rule.host = ingress.host;
    rule.http = new HTTPIngressRuleValue();
    rule.http.paths = ingress.getPaths().map<HTTPIngressPath>((path) => {
      const p = new HTTPIngressPath();
      p.path = path;
      p.pathType = "Prefix";

      p.backend = new IngressBackend();
      p.backend.service = new IngressServiceBackend();
      p.backend.service.name = name;
      p.backend.service.port = new ServiceBackendPort();
      p.backend.service.port.name = "http";
      return p;
    });

    return rule;
  });

  const i = Suffiks.getResource<K8sIngress>(
    "networking.k8s.io",
    "v1",
    "ingresses",
    name
  );
  if (!i.error) {
    spec.metadata.resourceVersion = i.resource!.metadata.resourceVersion;
    const update = Suffiks.updateResource<K8sIngress>(
      "networking.k8s.io",
      "v1",
      "ingresses",
      JSON.stringify(spec)
    );

    if (update.error) {
      console.error("Error updating ingress: " + update.error!.toString());
      return;
    }
    return;
  }

  if (i.error && !i.error!.isNotFound()) {
    console.error("Error getting ingress: " + i.error!.toString());
    return;
  }

  const res = Suffiks.createResource<K8sIngress>(
    "networking.k8s.io",
    "v1",
    "ingresses",
    JSON.stringify(spec)
  );

  if (res.error && !res.error!.isAlreadyExists()) {
    console.error("Error creating ingress: " + res.error!.toString());
    return;
  } else if (res.resource) {
    console.log("Created ingress: " + res.resource!.metadata.name);
    return;
  }
}

export function Delete(): void {
  const owner = Suffiks.getOwner();

  const err = Suffiks.deleteResource(
    "networking.k8s.io",
    "v1",
    "ingresses",
    owner.name
  );

  if (err) {
    console.error("Error deleting ingress: " + err.toString());
    return;
  }
}

function validateHost(host: string): boolean {
  if (!process.env.has("INGRESSES")) {
    console.log("NO CONFIGURATION FOUND");
    return true;
  }

  const hostsList = process.env.get("INGRESSES");
  if (!hostsList) {
    return true;
  }

  const hosts = hostsList.split(",");

  const validateHost = (valid: string, incoming: string): boolean => {
    const vp = valid.split(".");
    const ip = incoming.split(".");

    if (vp.length != ip.length) {
      return false;
    }

    for (let i = 0; i < vp.length; i++) {
      if (vp[i] != "*" && vp[i] != ip[i]) {
        return false;
      }
    }

    return true;
  };

  for (let i = 0; i < hosts.length; i++) {
    if (validateHost(hosts[i], host)) {
      return true;
    }
  }

  return false;
}
