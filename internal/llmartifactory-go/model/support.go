package model

import (
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

type Support struct {
	BuiltinRoot          rootModel.RootID
	BuiltinPackageSource sourceModel.SourceID
	ManagedSource        support.SourceProfile
	ProviderPackage      support.PackageLayout
	ModelPackage         support.PackageLayout
}

func (s Support) Validate() error {
	if err := s.BuiltinRoot.Validate(); err != nil {
		return err
	}
	if err := s.BuiltinPackageSource.Validate(); err != nil {
		return err
	}
	if err := s.ManagedSource.Validate(); err != nil {
		return err
	}
	if err := s.ProviderPackage.Validate(); err != nil {
		return err
	}
	return s.ModelPackage.Validate()
}

func (s Support) Clone() Support {
	output := s
	output.ManagedSource = s.ManagedSource.Clone()
	return output
}
