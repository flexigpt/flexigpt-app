package root

import rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"

// Policy identifies Roots whose source-backed topology may be changed only by
// trusted in-process installer work. It is behavioral policy rather than
// persisted Root data.
type Policy interface {
	IsProtectedRoot(rootID rootModel.RootID) bool
}

// DeletionPolicy optionally identifies Roots that must remain present.
// Retention is deliberately independent from protected-topology mutation.
type DeletionPolicy interface {
	IsRootDeletionProtected(rootID rootModel.RootID) bool
}
