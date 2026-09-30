package main

import (
	"context"

	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
)

type SettingStoreWrapper struct {
	store *settingStore.SettingStore
}

func InitSettingStoreWrapper(
	wrapper *SettingStoreWrapper,
	baseDir string,
) error {
	if wrapper == nil {
		panic("initialising SettingStoreWrapper with nil receiver")
	}

	store, err := settingStore.NewSettingStore(baseDir)
	if err != nil {
		return err
	}
	wrapper.store = store
	return nil
}

func (w *SettingStoreWrapper) SetAppTheme(
	request *settingSpec.SetAppThemeRequest,
) (*settingSpec.SetAppThemeResponse, error) {
	return withRecoveryResp(func() (*settingSpec.SetAppThemeResponse, error) {
		if w == nil || w.store == nil {
			return nil, settingStoreClosedError()
		}
		return w.store.SetAppTheme(context.Background(), request)
	})
}

func (w *SettingStoreWrapper) SetDebugSettings(
	request *settingSpec.SetDebugSettingsRequest,
) (*settingSpec.SetDebugSettingsResponse, error) {
	return withRecoveryResp(func() (*settingSpec.SetDebugSettingsResponse, error) {
		if w == nil || w.store == nil {
			return nil, settingStoreClosedError()
		}
		return w.store.SetDebugSettings(context.Background(), request)
	})
}

func (w *SettingStoreWrapper) GetSettings(
	request *settingSpec.GetSettingsRequest,
) (*settingSpec.GetSettingsResponse, error) {
	return withRecoveryResp(func() (*settingSpec.GetSettingsResponse, error) {
		if w == nil || w.store == nil {
			return nil, settingStoreClosedError()
		}
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

func settingStoreClosedError() error {
	return settingStore.ErrClosed()
}
