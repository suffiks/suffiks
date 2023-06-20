- [ ] Support updating extensions. (Is this just reconnect when connection is lost?)
- [ ] Retry if extension GRPC connection is closed
- [ ] Rescue from panics in extensions.
- [ ] Remove `Status.Hash` requirement. Possibly allowing extensions to set statuses etc as well.
- [ ] Status and metrics support for extensions
- [ ] Support for reloading WASI extensions when the version changes.
- [ ] Use `log/slog` for logging.
- [ ] Support for returning errors from WASI.

Probably not:

- [ ] Ensure extension service is available before addding it?
- [ ] Consider moving extensions into it's own module (reducing number of dependencies).
