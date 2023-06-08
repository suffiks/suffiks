export function Malloc(size: usize): usize {
  return heap.alloc(size);
}

export function Free(ptr: usize): void {
  heap.free(ptr);
}
