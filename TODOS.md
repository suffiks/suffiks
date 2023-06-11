- [x] Sync concurrently with extension services.
- [ ] Support updating extensions. (Is this just reconnect when connection is lost?)
- [ ] Retry if extension GRPC connection is closed
- [ ] Rescue from panics in extensions.
- [ ] Remove `Status.Hash` requirement. Possibly allowing extensions to set statuses etc as well.
- [ ] Status and metrics support for extensions

Probably not:

- [ ] Ensure extension service is available before addding it?
- [ ] Consider moving extensions into it's own module (reducing number of dependencies).
