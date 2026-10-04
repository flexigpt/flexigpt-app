# Work Package A caller migration

This delivery removes active compatibility façades. Update callers as follows:

| Previous import/use                                         | Replacement                                                                                                                                  |
| ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `store/root/impl.NewService`                                | `root.NewService`                                                                                                                            |
| `store/source/impl.NewRegistry`, `NewRuntime`, `NewService` | `source.NewRegistry`, `source.NewRuntime`, `source.NewService`                                                                               |
| `store/source/impl` snapshot helpers                        | `source.ReadSnapshotEntry`, `source.ReadVerifiedSnapshotEntry`, `source.VerifySnapshotContentDigest`, and `source.ResolveVerifiedLocalPath`  |
| `store/source/ingest/impl` engine/staging values            | `ingest.NewDecoderRegistry`, `ingest.NewEngine`, `ingest.Scanner`, `ingest.Result`, and `ingest.Observation`                                 |
| `store/artifact/impl.NewService`, `NewSynchronizer`         | `artifact.NewService`, `artifact.NewSynchronizer`                                                                                            |
| `artifact/idprovider`                                       | `artifact.IDProvider` and `artifact.NewUUIDIDProvider`                                                                                       |
| `refresh/impl.NewService`                                   | `refresh.NewService`                                                                                                                         |
| `resource/impl.NewService`                                  | `resource.NewService`; retain ordinary `resource.API` separately from `resource.NativePathAPI`                                               |
| `managepackage/compose.Open`                                | `managepackage.NewService(managepackage.Config{...})`                                                                                        |
| `install/compose.Open`                                      | `install.NewService(install.Config{...})`                                                                                                    |
| `install.WithPrivilege` and related helpers                 | `root.WithInstallerPrivilege`, `root.RequireInstallerPrivilege`, `root.IsInstallerPrivileged`                                                |
| `sqlite.LocalState()`                                       | `sqlite.Overlays()`, `sqlite.Secrets()`, and `sqlite.ArtifactCleanup()`                                                                      |
| one combined local-state service                            | `overlay.NewService`, `secret.NewBindingService`, `secret.NewRuntimeService`, `secret.NewLifecycleService`, and `artifactcleanup.NewService` |
| a concrete SQLite Definition repository as consumer API     | `definition.NewService(sqlite.Definitions())`                                                                                                |
| catalog complete-Artifact `GetMany`                         | `artifact.API.GetMany`; use `catalog.API` only for metadata catalog reads                                                                    |
| `AllowProtected` package request flags                      | remove them; protected Root mutation requires a trusted Root installer context                                                               |

`compose/local` still exists in this Work Package because aggregate assembly collapse is Work Package B. It now uses the named owner constructors above.
