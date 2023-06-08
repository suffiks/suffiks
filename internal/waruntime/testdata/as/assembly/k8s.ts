import { JSON } from "json-as";

function forceImport(): void {
  JSON.stringify("");
}

@json
class Owner {
  apiVersion!: string;
  kind!: string;
  name!: string;
  uid!: string;
  controller!: boolean;
}

@json
class ObjectReference {
  name!: string;
  namespace!: string;
  ownerReferences!: Owner[];
  resourceVersion: string | null;
}

@json
class IngressSpec {
  ingressClassName!: string;
  rules!: IngressRule[];
}

@json
class IngressRule {
  host!: string;
  http!: HTTPIngressRuleValue;
}

@json
class HTTPIngressRuleValue {
  paths!: HTTPIngressPath[];
}

@json
class HTTPIngressPath {
  path!: string;
  backend!: IngressBackend;
}

@json
class IngressBackend {
  service!: IngressServiceBackend;
}

@json
class IngressServiceBackend {
  name!: string;
  port!: ServiceBackendPort;
}

@json
class ServiceBackendPort {
  name!: string;
}

@json
class Ingress {
  apiVersion!: string;
  kind!: string;
  metadata!: ObjectReference;
  spec!: IngressSpec;
}

export {
  Ingress,
  IngressSpec,
  IngressRule,
  HTTPIngressRuleValue,
  HTTPIngressPath,
  IngressBackend,
  IngressServiceBackend,
  ServiceBackendPort,
  ObjectReference,
  Owner,
};
