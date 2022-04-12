- [x] Sync concurrently with extension services.
- [ ] Support updating extensions. (Is this just reconnect when connection is lost?)
- [ ] Retry if extension GRPC connection is closed
- [ ] Rescue from panics in extensions.
- [x] Configuration using [component config](https://book.kubebuilder.io/component-config-tutorial/tutorial.html)?
- [ ] Remove `Status.Hash` requirement. Possibly allowing extensions to set statuses etc as well.
- [x] Configure extensions using something similar to component config
- [ ] Status and metrics support for extensions

Probably not:

- [ ] Ensure extension service is available before addding it?
- [ ] Consider moving extensions into it's own module (reducing number of dependencies).
