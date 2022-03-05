- [x] Sync concurrently with extension services.
- [ ] Support updating extensions.
- [ ] Rescue from panics in extensions.
- [ ] Retry if extension GRPC connection is closed
- [x] Configuration using [component config](https://book.kubebuilder.io/component-config-tutorial/tutorial.html)?
- [ ] Remove `Status.Hash` requirement. Possibly allowing extensions to set statuses etc as well.
- [ ] Configure extensions using something similar to component config
- [ ] Status and metrics support for extensions

Probably not:
- [ ] Ensure extension service is available before addding it?
- [ ] Consider moving extensions into it's own module (reducing number of dependencies).
