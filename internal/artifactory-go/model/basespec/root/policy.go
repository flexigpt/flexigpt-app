package root

// RootPolicy identifies Roots whose source, package, and ordinary Artifact
// mutations are prohibited. The application composition owns the concrete
// policy and the protected-root declaration. Universal Artifact enablement is
// intentionally outside this policy because it is local metadata rather than
// protected source mutation.
type RootPolicy interface {
	IsProtectedRoot(r RootID) bool
}

// RootDeletionPolicy is an optional lifecycle policy for Roots that must
// remain present while still allowing ordinary descendant mutation.
//
// A retained Root is intentionally different from a protected topology Root:
// protected Roots reject ordinary Source and Artifact mutations; retained
// Roots reject only Root retirement and purge.
type RootDeletionPolicy interface {
	IsRootDeletionProtected(rootID RootID) bool
}
