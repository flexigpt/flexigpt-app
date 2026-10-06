package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/setting"
)

type SettingStoreWrapper struct {
	store *setting.SettingStore
}

func InitSettingStoreWrapper(
	wrapper *SettingStoreWrapper,
	baseDir string,
) error {
	if wrapper == nil {
		panic("initialising SettingStoreWrapper with nil receiver")
	}

	store, err := setting.NewSettingStore(baseDir)
	if err != nil {
		return err
	}
	wrapper.store = store
	return nil
}

func (w *SettingStoreWrapper) SetAppTheme(
	request *setting.SetAppThemeRequest,
) (*setting.SetAppThemeResponse, error) {
	return withRecoveryResp(func() (*setting.SetAppThemeResponse, error) {
		return w.store.SetAppTheme(context.Background(), request)
	})
}

func (w *SettingStoreWrapper) SetDebugSettings(
	request *setting.SetDebugSettingsRequest,
) (*setting.SetDebugSettingsResponse, error) {
	return withRecoveryResp(func() (*setting.SetDebugSettingsResponse, error) {
		return w.store.SetDebugSettings(context.Background(), request)
	})
}

func (w *SettingStoreWrapper) GetSettings(
	request *setting.GetSettingsRequest,
) (*setting.GetSettingsResponse, error) {
	return withRecoveryResp(func() (*setting.GetSettingsResponse, error) {
		return w.store.GetSettings(context.Background(), request)
	})
}

func (w *SettingStoreWrapper) close() {
	if w == nil || w.store == nil {
		return
	}
	_ = w.store.Close()
	w.store = nil
}
