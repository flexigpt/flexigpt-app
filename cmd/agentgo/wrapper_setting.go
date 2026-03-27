package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"

	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
)

type SettingStoreWrapper struct {
	store *settingStore.SettingStore
}

// InitSettingStoreWrapper boots the underlying store and remembers the pointer.
func InitSettingStoreWrapper(
	w *SettingStoreWrapper,
	baseDir string,
) error {
	if w == nil {
		panic("initialising SettingStoreWrapper with <nil> receivers")
	}
	ss, err := settingStore.NewSettingStore(baseDir)
	if err != nil {
		return err
	}
	w.store = ss

	return nil
}

func (w *SettingStoreWrapper) SetAppTheme(
	req *settingSpec.SetAppThemeRequest,
) (*settingSpec.SetAppThemeResponse, error) {
	return middleware.WithRecoveryResp(func() (*settingSpec.SetAppThemeResponse, error) {
		return w.store.SetAppTheme(context.Background(), req)
	})
}

func (w *SettingStoreWrapper) SetDebugSettings(
	req *settingSpec.SetDebugSettingsRequest,
) (*settingSpec.SetDebugSettingsResponse, error) {
	return middleware.WithRecoveryResp(func() (*settingSpec.SetDebugSettingsResponse, error) {
		return w.store.SetDebugSettings(context.Background(), req)
	})
}

func (w *SettingStoreWrapper) GetSettings(
	req *settingSpec.GetSettingsRequest,
) (*settingSpec.GetSettingsResponse, error) {
	return middleware.WithRecoveryResp(func() (*settingSpec.GetSettingsResponse, error) {
		return w.store.GetSettings(context.Background(), req)
	})
}

func (w *SettingStoreWrapper) GetAuthKey(
	req *settingSpec.GetAuthKeyRequest,
) (*settingSpec.GetAuthKeyResponse, error) {
	return middleware.WithRecoveryResp(func() (*settingSpec.GetAuthKeyResponse, error) {
		return w.store.GetAuthKey(context.Background(), req)
	})
}
