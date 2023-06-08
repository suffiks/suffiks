export namespace Host {
	// @ts-ignore decorators are valid here
	@external("suffiks", "AddEnv")
	export declare function addEnv(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "AddEnvFrom")
	export declare function addEnvFrom(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "AddLabel")
	export declare function addLabel(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "AddAnnotation")
	export declare function addAnnotation(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "AddInitContainer")
	export declare function addInitContainer(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "AddSidecar")
	export declare function addSidecar(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "MergePatch")
	export declare function mergePatch(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "ValidationError")
	export declare function validationError(ptr : u32, size : u32): void;

	// @ts-ignore decorators are valid here
	@external("suffiks", "GetOwner")
	export declare function getOwner():  u32;

	// @ts-ignore decorators are valid here
	@external("suffiks", "GetSpec")
	export declare function getSpec():  u32;

	// @ts-ignore decorators are valid here
	@external("suffiks", "GetOld")
	export declare function getOld():  u32;

	// @ts-ignore decorators are valid here
	@external("suffiks", "CreateResource")
	export declare function createResource(gvrPtr : u32, gvrSize : u32, specPtr : u32, specSize: u32): u32;

	// @ts-ignore decorators are valid here
	@external("suffiks", "UpdateResource")
	export declare function updateResource(gvrPtr : u32, gvrSize : u32, specPtr : u32, specSize: u32): u32;

	// @ts-ignore decorators are valid here
	@external("suffiks", "GetResource")
	export declare function getResource(ptr : u32, size : u32, namePtr : u32, nameSize : u32): u32;

	// @ts-ignore decorators are valid here
	@external("suffiks", "DeleteResource")
	export declare function deleteResource(ptr : u32, size : u32): u32;
}
